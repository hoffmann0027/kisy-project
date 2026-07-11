//go:build integration

package scheduled_test

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"kisy-backend/internal/audit"
	"kisy-backend/internal/messages"
	"kisy-backend/internal/platform/testdb"
	"kisy-backend/internal/scheduled"
)

// harness wires a real messages.Service behind the scheduled service, with
// switchable access/user stubs so tests can revoke access between
// scheduling and send time.
type harness struct {
	svc      *scheduled.Service
	msgs     *messages.Service
	pool     *pgxpool.Pool
	denied   *atomic.Bool // deny all chat access when set
	inactive *atomic.Bool // report the sender deactivated when set
	noAttach *atomic.Bool // report every attachment gone when set
	alice    scheduled.Actor
	chat     uuid.UUID
	ctx      context.Context
}

func setup(t *testing.T) *harness {
	pool := testdb.New(t)
	ctx := context.Background()

	alice := testdb.SeedUser(t, pool, "alice", 3)
	chat := uuid.New()

	denied := &atomic.Bool{}
	inactive := &atomic.Bool{}
	noAttach := &atomic.Bool{}

	authorize := func(_ context.Context, _ string, _ uuid.UUID, _ uuid.UUID, _ int) error {
		if denied.Load() {
			return errors.New("no access")
		}
		return nil
	}

	rec := audit.NewPostgresRecorder(slog.New(slog.NewTextHandler(io.Discard, nil)))
	msgs := messages.NewService(pool, messages.NewPostgresRepository(), rec, messages.Authorizer{
		Private: func(_ context.Context, _ uuid.UUID, _ uuid.UUID) error {
			if denied.Load() {
				return messages.ErrNotFound
			}
			return nil
		},
		Group: func(_ context.Context, _ uuid.UUID, _ uuid.UUID, _ int) error {
			if denied.Load() {
				return messages.ErrNotFound
			}
			return nil
		},
	})

	svc := scheduled.NewService(pool, scheduled.NewPostgresRepository(),
		authorize,
		func(_ context.Context, _ uuid.UUID) (int, bool, error) {
			return 3, !inactive.Load(), nil
		},
		func(_ context.Context, ids []uuid.UUID, _ uuid.UUID) ([]uuid.UUID, error) {
			if noAttach.Load() {
				return nil, nil
			}
			return ids, nil
		},
		msgs,
	)

	return &harness{
		svc: svc, msgs: msgs, pool: pool,
		denied: denied, inactive: inactive, noAttach: noAttach,
		alice: scheduled.Actor{UserID: alice, RoleLevel: 3},
		chat:  chat, ctx: ctx,
	}
}

// forceDue rewinds a pending row's send_at so the worker sees it as due.
func (h *harness) forceDue(t *testing.T, id uuid.UUID) {
	t.Helper()
	if _, err := h.pool.Exec(h.ctx, `UPDATE scheduled_messages SET send_at = now() - interval '1 second' WHERE id = $1`, id); err != nil {
		t.Fatalf("force due: %v", err)
	}
}

func (h *harness) schedule(t *testing.T, in scheduled.Input) scheduled.DTO {
	t.Helper()
	dto, err := h.svc.Schedule(h.ctx, in, h.alice)
	if err != nil {
		t.Fatalf("schedule: %v", err)
	}
	return dto
}

func textInput(chat uuid.UUID, text string) scheduled.Input {
	return scheduled.Input{
		ChatType: "private", ChatID: chat, Text: text,
		SendAt: time.Now().Add(time.Hour),
	}
}

func TestScheduleLifecycle(t *testing.T) {
	h := setup(t)

	dto := h.schedule(t, textInput(h.chat, "будущее сообщение"))
	if dto.Status != scheduled.StatusPending {
		t.Fatalf("status = %q", dto.Status)
	}

	// List shows it.
	list, err := h.svc.List(h.ctx, h.alice)
	if err != nil || len(list) != 1 || list[0].ID != dto.ID {
		t.Fatalf("list: %+v, %v", list, err)
	}

	// Edit text and time.
	newText := "отредактировано"
	newAt := time.Now().Add(2 * time.Hour)
	upd, err := h.svc.Update(h.ctx, dto.ID, scheduled.UpdateInput{Text: &newText, SendAt: &newAt}, h.alice)
	if err != nil || upd.Text == nil || *upd.Text != newText {
		t.Fatalf("update: %+v, %v", upd, err)
	}

	// A stranger can neither see nor touch it (masked 404).
	bob := scheduled.Actor{UserID: testdb.SeedUser(t, h.pool, "bob", 8), RoleLevel: 8}
	if list, _ := h.svc.List(h.ctx, bob); len(list) != 0 {
		t.Fatalf("bob sees alice's scheduled: %+v", list)
	}
	if _, err := h.svc.Update(h.ctx, dto.ID, scheduled.UpdateInput{SendAt: &newAt}, bob); !errors.Is(err, scheduled.ErrNotFound) {
		t.Fatalf("foreign update: want ErrNotFound, got %v", err)
	}
	if err := h.svc.Cancel(h.ctx, dto.ID, bob); !errors.Is(err, scheduled.ErrNotFound) {
		t.Fatalf("foreign cancel: want ErrNotFound, got %v", err)
	}

	// Cancel removes the row entirely.
	if err := h.svc.Cancel(h.ctx, dto.ID, h.alice); err != nil {
		t.Fatalf("cancel: %v", err)
	}
	if list, _ := h.svc.List(h.ctx, h.alice); len(list) != 0 {
		t.Fatalf("after cancel: %+v", list)
	}
}

func TestWorkerSendsExactlyOnce(t *testing.T) {
	h := setup(t)

	dto := h.schedule(t, textInput(h.chat, "точно один раз"))
	h.forceDue(t, dto.ID)

	// First pass sends.
	n, err := h.svc.ProcessDue(h.ctx, time.Now(), 50)
	if err != nil || n != 1 {
		t.Fatalf("first pass: n=%d, %v", n, err)
	}
	// Second pass finds nothing.
	n, err = h.svc.ProcessDue(h.ctx, time.Now(), 50)
	if err != nil || n != 0 {
		t.Fatalf("second pass: n=%d, %v", n, err)
	}

	// Exactly one message exists, stamped with the scheduled id.
	var count int
	var schedID uuid.UUID
	if err := h.pool.QueryRow(h.ctx, `
		SELECT COUNT(*), MIN(scheduled_message_id::text)::uuid FROM messages
		WHERE chat_id = $1 AND text = 'точно один раз'`, h.chat).Scan(&count, &schedID); err != nil {
		t.Fatalf("count: %v", err)
	}
	if count != 1 || schedID != dto.ID {
		t.Fatalf("count=%d schedID=%v (want 1, %v)", count, schedID, dto.ID)
	}

	// The row is marked sent and references the message.
	var status string
	var sentMsg *uuid.UUID
	if err := h.pool.QueryRow(h.ctx, `
		SELECT status, sent_message_id FROM scheduled_messages WHERE id = $1`, dto.ID).Scan(&status, &sentMsg); err != nil {
		t.Fatalf("row: %v", err)
	}
	if status != scheduled.StatusSent || sentMsg == nil {
		t.Fatalf("status=%q sentMsg=%v", status, sentMsg)
	}

	// A sent row can no longer be edited or canceled.
	at := time.Now().Add(time.Hour)
	if _, err := h.svc.Update(h.ctx, dto.ID, scheduled.UpdateInput{SendAt: &at}, h.alice); !errors.Is(err, scheduled.ErrNotFound) {
		t.Fatalf("update sent: want ErrNotFound, got %v", err)
	}
	if err := h.svc.Cancel(h.ctx, dto.ID, h.alice); !errors.Is(err, scheduled.ErrNotFound) {
		t.Fatalf("cancel sent: want ErrNotFound, got %v", err)
	}
}

func TestWorkerCancelsOnLostAccess(t *testing.T) {
	h := setup(t)

	dto := h.schedule(t, textInput(h.chat, "не должно уйти"))
	h.forceDue(t, dto.ID)

	// Access revoked between scheduling and send time.
	h.denied.Store(true)
	n, err := h.svc.ProcessDue(h.ctx, time.Now(), 50)
	if err != nil || n != 1 {
		t.Fatalf("pass: n=%d, %v", n, err)
	}

	var status string
	var text *string
	if err := h.pool.QueryRow(h.ctx, `SELECT status, text FROM scheduled_messages WHERE id = $1`, dto.ID).Scan(&status, &text); err != nil {
		t.Fatalf("row: %v", err)
	}
	if status != scheduled.StatusCanceled {
		t.Fatalf("status = %q, want canceled", status)
	}
	// Cancellation drops the frozen content.
	if text != nil {
		t.Fatalf("canceled row still holds text")
	}
	// Nothing was sent.
	var count int
	_ = h.pool.QueryRow(h.ctx, `SELECT COUNT(*) FROM messages WHERE chat_id = $1`, h.chat).Scan(&count)
	if count != 0 {
		t.Fatalf("message leaked after access loss")
	}
}

func TestWorkerCancelsDeactivatedSender(t *testing.T) {
	h := setup(t)

	dto := h.schedule(t, textInput(h.chat, "от деактивированного"))
	h.forceDue(t, dto.ID)

	h.inactive.Store(true)
	if _, err := h.svc.ProcessDue(h.ctx, time.Now(), 50); err != nil {
		t.Fatalf("pass: %v", err)
	}

	var status string
	_ = h.pool.QueryRow(h.ctx, `SELECT status FROM scheduled_messages WHERE id = $1`, dto.ID).Scan(&status)
	if status != scheduled.StatusCanceled {
		t.Fatalf("status = %q, want canceled", status)
	}
}

func TestWorkerE2EESnapshot(t *testing.T) {
	h := setup(t)

	alg := int16(1)
	epoch := int64(4)
	kind := int16(1)
	ciphertext := []byte{0xDE, 0xAD, 0xBE, 0xEF}
	dto := h.schedule(t, scheduled.Input{
		ChatType: "private", ChatID: h.chat,
		Ciphertext: ciphertext, Alg: &alg, Epoch: &epoch, ContentKind: &kind,
		SendAt: time.Now().Add(time.Hour),
	})
	h.forceDue(t, dto.ID)

	if _, err := h.svc.ProcessDue(h.ctx, time.Now(), 50); err != nil {
		t.Fatalf("pass: %v", err)
	}

	// The message carries the frozen ciphertext + epoch and the scheduled id.
	var gotCipher []byte
	var gotEpoch int64
	var gotSched uuid.UUID
	var gotText *string
	if err := h.pool.QueryRow(h.ctx, `
		SELECT ciphertext, epoch, scheduled_message_id, text FROM messages WHERE chat_id = $1`, h.chat).
		Scan(&gotCipher, &gotEpoch, &gotSched, &gotText); err != nil {
		t.Fatalf("message: %v", err)
	}
	if string(gotCipher) != string(ciphertext) || gotEpoch != epoch || gotSched != dto.ID || gotText != nil {
		t.Fatalf("snapshot mismatch: cipher=%x epoch=%d sched=%v text=%v", gotCipher, gotEpoch, gotSched, gotText)
	}
}

func TestWorkerCancelsWhenContentGone(t *testing.T) {
	h := setup(t)

	// Attachment-only snapshot whose files vanish before send time.
	dto := h.schedule(t, scheduled.Input{
		ChatType: "private", ChatID: h.chat,
		AttachmentIDs: []uuid.UUID{uuid.New()},
		SendAt:        time.Now().Add(time.Hour),
	})
	h.forceDue(t, dto.ID)

	h.noAttach.Store(true)
	if _, err := h.svc.ProcessDue(h.ctx, time.Now(), 50); err != nil {
		t.Fatalf("pass: %v", err)
	}

	var status string
	_ = h.pool.QueryRow(h.ctx, `SELECT status FROM scheduled_messages WHERE id = $1`, dto.ID).Scan(&status)
	if status != scheduled.StatusCanceled {
		t.Fatalf("status = %q, want canceled", status)
	}
	var count int
	_ = h.pool.QueryRow(h.ctx, `SELECT COUNT(*) FROM messages WHERE chat_id = $1`, h.chat).Scan(&count)
	if count != 0 {
		t.Fatalf("empty message leaked")
	}
}

func TestSchedulePendingCap(t *testing.T) {
	h := setup(t)
	// The cap rejects the 101st pending row. Insert 100 quickly via the
	// service (validation exercised each time).
	for i := 0; i < scheduled.MaxPending; i++ {
		h.schedule(t, textInput(h.chat, "n"))
	}
	if _, err := h.svc.Schedule(h.ctx, textInput(h.chat, "перебор"), h.alice); !errors.Is(err, scheduled.ErrValidation) {
		t.Fatalf("cap: want ErrValidation, got %v", err)
	}
}
