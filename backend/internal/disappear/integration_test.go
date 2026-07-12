//go:build integration

package disappear_test

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"kisy-backend/internal/audit"
	"kisy-backend/internal/disappear"
	"kisy-backend/internal/messages"
	"kisy-backend/internal/platform/db"
	"kisy-backend/internal/platform/testdb"
)

// fakePub records expiry fan-outs.
type fakePub struct {
	mu      sync.Mutex
	expired []uuid.UUID
}

func (p *fakePub) PublishMessageExpired(_ string, _ uuid.UUID, messageID uuid.UUID) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.expired = append(p.expired, messageID)
}

type harness struct {
	svc    *disappear.Service
	msgs   *messages.Service
	pool   *pgxpool.Pool
	pub    *fakePub
	alice  disappear.Actor
	mAlice messages.ActorMeta
	chat   uuid.UUID
	ctx    context.Context
}

func setup(t *testing.T) *harness {
	pool := testdb.New(t)
	ctx := context.Background()
	alice := testdb.SeedUser(t, pool, "alice", 3)
	chat := uuid.New()

	allow := func(_ context.Context, _ string, _ uuid.UUID, _ uuid.UUID, _ int) error { return nil }
	rec := audit.NewPostgresRecorder(slog.New(slog.NewTextHandler(io.Discard, nil)))

	msgs := messages.NewService(pool, messages.NewPostgresRepository(), rec, messages.Authorizer{
		Private: func(_ context.Context, _ uuid.UUID, _ uuid.UUID) error { return nil },
		Group:   func(_ context.Context, _ uuid.UUID, _ uuid.UUID, _ int) error { return nil },
	})
	// Forwarding needs a clearance resolver; both chats share one breadth here.
	msgs.SetForwarding(messages.ClearanceResolver{
		Private: func(_ context.Context, _ uuid.UUID) (int, error) { return 5, nil },
		Group:   func(_ context.Context, _ uuid.UUID) (int, error) { return 5, nil },
	}, nil, nil)

	pub := &fakePub{}
	svc := disappear.NewService(pool, disappear.NewPostgresRepository(), allow, rec)
	svc.SetPublisher(pub)
	msgs.SetDisappearTTL(svc.TTLFor)

	return &harness{
		svc: svc, msgs: msgs, pool: pool, pub: pub,
		alice:  disappear.Actor{UserID: alice, RoleLevel: 3},
		mAlice: messages.ActorMeta{UserID: alice, RoleLevel: 3},
		chat:   chat, ctx: ctx,
	}
}

func (h *harness) send(t *testing.T, text string, ttl *int64) messages.DTO {
	t.Helper()
	dto, err := h.msgs.Send(h.ctx, messages.SendInput{
		ChatType: "private", ChatID: h.chat, Text: text, TTLSeconds: ttl,
	}, h.mAlice)
	if err != nil {
		t.Fatalf("send: %v", err)
	}
	return dto
}

func (h *harness) forceExpired(t *testing.T, id uuid.UUID) {
	t.Helper()
	if _, err := h.pool.Exec(h.ctx, `UPDATE messages SET expires_at = now() - interval '1 second' WHERE id = $1`, id); err != nil {
		t.Fatalf("force expired: %v", err)
	}
}

func TestChatTTLSettingLifecycle(t *testing.T) {
	h := setup(t)
	ttl := int64(3600)

	// Off by default.
	s, err := h.svc.Get(h.ctx, "private", h.chat, h.alice)
	if err != nil || s.TTLSeconds != nil {
		t.Fatalf("default: %+v, %v", s, err)
	}

	// Enable → new messages get expires_at; message sent before stays clean.
	before := h.send(t, "до таймера", nil)
	if before.ExpiresAt != nil {
		t.Fatalf("pre-TTL message should not expire")
	}
	if _, err := h.svc.Set(h.ctx, "private", h.chat, &ttl, h.alice); err != nil {
		t.Fatalf("set: %v", err)
	}
	after := h.send(t, "после таймера", nil)
	if after.ExpiresAt == nil {
		t.Fatalf("post-TTL message should expire")
	}
	want := time.Now().Add(time.Duration(ttl) * time.Second)
	if d := after.ExpiresAt.Sub(want); d < -time.Minute || d > time.Minute {
		t.Fatalf("expires_at off by %v", d)
	}

	// The audit trail records the change (TTL only).
	var n int
	_ = h.pool.QueryRow(h.ctx, `SELECT COUNT(*) FROM audit_logs WHERE action = $1`, disappear.ActionDisappearingSet).Scan(&n)
	if n != 1 {
		t.Fatalf("audit rows = %d, want 1", n)
	}

	// Disable → back to non-expiring.
	if _, err := h.svc.Set(h.ctx, "private", h.chat, nil, h.alice); err != nil {
		t.Fatalf("clear: %v", err)
	}
	final := h.send(t, "снова без таймера", nil)
	if final.ExpiresAt != nil {
		t.Fatalf("post-clear message should not expire")
	}

	// Validation bounds.
	bad := int64(1)
	if _, err := h.svc.Set(h.ctx, "private", h.chat, &bad, h.alice); !errors.Is(err, disappear.ErrValidation) {
		t.Fatalf("too-short TTL: want ErrValidation, got %v", err)
	}
}

func TestPerMessageTTLOverridesChat(t *testing.T) {
	h := setup(t)
	chatTTL := int64(86400)
	if _, err := h.svc.Set(h.ctx, "private", h.chat, &chatTTL, h.alice); err != nil {
		t.Fatalf("set: %v", err)
	}
	msgTTL := int64(60)
	dto := h.send(t, "короткоживущее", &msgTTL)
	if dto.ExpiresAt == nil {
		t.Fatalf("no expiry")
	}
	if dto.ExpiresAt.After(time.Now().Add(2 * time.Minute)) {
		t.Fatalf("per-message TTL should win over chat TTL: %v", dto.ExpiresAt)
	}
}

func TestSetExpiryOnOwnMessage(t *testing.T) {
	h := setup(t)
	dto := h.send(t, "потом исчезну", nil)

	ttl := int64(300)
	upd, err := h.msgs.SetExpiry(h.ctx, dto.ID, &ttl, h.mAlice)
	if err != nil || upd.ExpiresAt == nil {
		t.Fatalf("set expiry: %+v, %v", upd, err)
	}

	// A stranger cannot set a timer on someone else's message.
	bob := messages.ActorMeta{UserID: testdb.SeedUser(t, h.pool, "bob", 8), RoleLevel: 8}
	if _, err := h.msgs.SetExpiry(h.ctx, dto.ID, &ttl, bob); !errors.Is(err, messages.ErrForbidden) {
		t.Fatalf("foreign set expiry: want ErrForbidden, got %v", err)
	}

	// Clearing works.
	cleared, err := h.msgs.SetExpiry(h.ctx, dto.ID, nil, h.mAlice)
	if err != nil || cleared.ExpiresAt != nil {
		t.Fatalf("clear expiry: %+v, %v", cleared, err)
	}
}

func TestReaperHardDeletesExpired(t *testing.T) {
	h := setup(t)

	// One expiring message with an attachment, one that stays.
	ttl := int64(3600)
	doomed := h.send(t, "исчезающее", &ttl)
	keeper := h.send(t, "вечное", nil)

	var attID uuid.UUID
	if err := h.pool.QueryRow(h.ctx, `
		INSERT INTO attachments (message_id, file_name, mime_type, size_bytes, storage_path, scan_status, data, uploaded_by)
		VALUES ($1, 'секрет.txt', 'text/plain', 6, '', 'clean', 'секрет', $2) RETURNING id`,
		doomed.ID, h.alice.UserID).Scan(&attID); err != nil {
		t.Fatalf("attach: %v", err)
	}

	h.forceExpired(t, doomed.ID)

	n, err := h.svc.ProcessExpired(h.ctx, time.Now(), 100)
	if err != nil || n != 1 {
		t.Fatalf("reap: n=%d, %v", n, err)
	}
	// Idempotent: nothing left to reap.
	n, _ = h.svc.ProcessExpired(h.ctx, time.Now(), 100)
	if n != 0 {
		t.Fatalf("second pass reaped %d", n)
	}

	// The row AND its attachment bytes are gone; the keeper survives.
	assertCount := func(q string, args []any, want int, label string) {
		t.Helper()
		var got int
		if err := h.pool.QueryRow(h.ctx, q, args...).Scan(&got); err != nil {
			t.Fatalf("%s: %v", label, err)
		}
		if got != want {
			t.Fatalf("%s = %d, want %d", label, got, want)
		}
	}
	assertCount(`SELECT COUNT(*) FROM messages WHERE id = $1`, []any{doomed.ID}, 0, "doomed message")
	assertCount(`SELECT COUNT(*) FROM attachments WHERE id = $1`, []any{attID}, 0, "attachment")
	assertCount(`SELECT COUNT(*) FROM messages WHERE id = $1`, []any{keeper.ID}, 1, "keeper message")

	// Clients were told exactly once.
	h.pub.mu.Lock()
	defer h.pub.mu.Unlock()
	if len(h.pub.expired) != 1 || h.pub.expired[0] != doomed.ID {
		t.Fatalf("fan-out: %v", h.pub.expired)
	}
}

func TestForwardInheritsTargetTTL(t *testing.T) {
	h := setup(t)

	// Source chat has no timer; target chat does.
	target := uuid.New()
	ttl := int64(3600)
	if _, err := h.svc.Set(h.ctx, "private", target, &ttl, h.alice); err != nil {
		t.Fatalf("set target ttl: %v", err)
	}
	src := h.send(t, "перешлют меня", nil)

	out, err := h.msgs.Forward(h.ctx, messages.ForwardInput{
		SourceMessageIDs: []uuid.UUID{src.ID},
		TargetChatType:   "private",
		TargetChatID:     target,
	}, h.mAlice)
	if err != nil || len(out) != 1 {
		t.Fatalf("forward: %v, %v", out, err)
	}
	if out[0].ExpiresAt == nil {
		t.Fatalf("forward should inherit the TARGET chat's TTL")
	}
	// And the source stays non-expiring.
	if src.ExpiresAt != nil {
		t.Fatalf("source gained a timer")
	}
}

// The scheduled worker path: a message sent through SendTx into a
// disappearing chat gets expires_at from SEND time, not scheduling time.
func TestSendTxAppliesChatTTL(t *testing.T) {
	h := setup(t)
	ttl := int64(600)
	if _, err := h.svc.Set(h.ctx, "private", h.chat, &ttl, h.alice); err != nil {
		t.Fatalf("set: %v", err)
	}

	tx, err := h.pool.Begin(h.ctx)
	if err != nil {
		t.Fatalf("begin: %v", err)
	}
	dto, deliver, err := h.msgs.SendTx(h.ctx, db.DBTX(tx), messages.SendInput{
		ChatType: "private", ChatID: h.chat, Text: "из планировщика",
	}, h.mAlice)
	if err != nil {
		t.Fatalf("sendtx: %v", err)
	}
	if err := tx.Commit(h.ctx); err != nil {
		t.Fatalf("commit: %v", err)
	}
	deliver(h.ctx)
	if dto.ExpiresAt == nil {
		t.Fatalf("SendTx should stamp the chat TTL")
	}
}
