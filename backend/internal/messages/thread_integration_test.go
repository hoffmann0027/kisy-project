//go:build integration

package messages_test

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"kisy-backend/internal/audit"
	"kisy-backend/internal/disappear"
	"kisy-backend/internal/messages"
	"kisy-backend/internal/platform/testdb"
)

// threadHarness wires a messages service where alice is a member of one
// group and mallory of nothing (denied everywhere).
type threadHarness struct {
	svc     *messages.Service
	pool    *pgxpool.Pool
	group   uuid.UUID
	alice   messages.ActorMeta
	mallory messages.ActorMeta
	ctx     context.Context
}

func threadSetup(t *testing.T) *threadHarness {
	pool := testdb.New(t)
	ctx := context.Background()
	alice := testdb.SeedUser(t, pool, "alice", 3)
	mallory := testdb.SeedUser(t, pool, "mallory", 9)
	group := uuid.New()

	rec := audit.NewPostgresRecorder(slog.New(slog.NewTextHandler(io.Discard, nil)))
	svc := messages.NewService(pool, messages.NewPostgresRepository(), rec, messages.Authorizer{
		Private: func(_ context.Context, _ uuid.UUID, actorID uuid.UUID) error {
			if actorID == mallory {
				return messages.ErrNotFound
			}
			return nil
		},
		Group: func(_ context.Context, _ uuid.UUID, actorID uuid.UUID, _ int) error {
			if actorID == mallory {
				return messages.ErrNotFound
			}
			return nil
		},
	})

	return &threadHarness{
		svc: svc, pool: pool, group: group,
		alice:   messages.ActorMeta{UserID: alice, RoleLevel: 3},
		mallory: messages.ActorMeta{UserID: mallory, RoleLevel: 9},
		ctx:     ctx,
	}
}

func (h *threadHarness) send(t *testing.T, text string, threadRoot *uuid.UUID) messages.DTO {
	t.Helper()
	dto, err := h.svc.Send(h.ctx, messages.SendInput{
		ChatType: "group", ChatID: h.group, Text: text, ThreadRootID: threadRoot,
	}, h.alice)
	if err != nil {
		t.Fatalf("send %q: %v", text, err)
	}
	return dto
}

func TestThreadRepliesAndCounters(t *testing.T) {
	h := threadSetup(t)

	root := h.send(t, "корень обсуждения", nil)
	h.send(t, "первый ответ", &root.ID)
	reply2 := h.send(t, "второй ответ", &root.ID)
	if reply2.ThreadRootID == nil || *reply2.ThreadRootID != root.ID {
		t.Fatalf("reply carries no root: %+v", reply2)
	}

	// The main feed contains only the root, with live counters.
	page, err := h.svc.List(h.ctx, "group", h.group, "", 50, h.alice)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(page.Items) != 1 || page.Items[0].ID != root.ID {
		t.Fatalf("main feed should hold only the root, got %d items", len(page.Items))
	}
	got := page.Items[0]
	if got.ThreadReplyCount != 2 || got.ThreadLastReplyAt == nil {
		t.Fatalf("root counters: count=%d lastReply=%v", got.ThreadReplyCount, got.ThreadLastReplyAt)
	}
	if time.Since(*got.ThreadLastReplyAt) > time.Minute {
		t.Fatalf("thread_last_reply_at stale: %v", got.ThreadLastReplyAt)
	}

	// The thread page returns both replies and nothing else.
	thread, err := h.svc.ListThread(h.ctx, root.ID, "", 50, h.alice)
	if err != nil {
		t.Fatalf("thread: %v", err)
	}
	if len(thread.Items) != 2 {
		t.Fatalf("thread items = %d, want 2", len(thread.Items))
	}
	for _, it := range thread.Items {
		if it.ThreadRootID == nil || *it.ThreadRootID != root.ID {
			t.Fatalf("stray message in thread: %+v", it)
		}
	}
}

func TestThreadValidation(t *testing.T) {
	h := threadSetup(t)
	root := h.send(t, "корень", nil)
	reply := h.send(t, "ответ", &root.ID)

	// No nested threads: replying "into" a reply is rejected.
	if _, err := h.svc.Send(h.ctx, messages.SendInput{
		ChatType: "group", ChatID: h.group, Text: "вложенный", ThreadRootID: &reply.ID,
	}, h.alice); !errors.Is(err, messages.ErrForbidden) {
		t.Fatalf("nested thread: want ErrForbidden, got %v", err)
	}

	// Threads are group-only.
	if _, err := h.svc.Send(h.ctx, messages.SendInput{
		ChatType: "private", ChatID: uuid.New(), Text: "в личке", ThreadRootID: &root.ID,
	}, h.alice); !errors.Is(err, messages.ErrForbidden) {
		t.Fatalf("private thread: want ErrForbidden, got %v", err)
	}

	// The root must live in the same chat (foreign roots are masked 404).
	otherGroup := uuid.New()
	if _, err := h.svc.Send(h.ctx, messages.SendInput{
		ChatType: "group", ChatID: otherGroup, Text: "чужой корень", ThreadRootID: &root.ID,
	}, h.alice); !errors.Is(err, messages.ErrNotFound) {
		t.Fatalf("cross-chat root: want ErrNotFound, got %v", err)
	}

	// Unknown root.
	missing := uuid.New()
	if _, err := h.svc.Send(h.ctx, messages.SendInput{
		ChatType: "group", ChatID: h.group, Text: "в никуда", ThreadRootID: &missing,
	}, h.alice); !errors.Is(err, messages.ErrNotFound) {
		t.Fatalf("missing root: want ErrNotFound, got %v", err)
	}
}

func TestThreadAccessMasked(t *testing.T) {
	h := threadSetup(t)
	root := h.send(t, "секретное обсуждение", nil)
	h.send(t, "ответ", &root.ID)

	// A non-member can neither read the thread nor post into it — both
	// surface as not-found, never confirming the root exists.
	if _, err := h.svc.ListThread(h.ctx, root.ID, "", 50, h.mallory); !errors.Is(err, messages.ErrNotFound) {
		t.Fatalf("foreign thread read: want ErrNotFound, got %v", err)
	}
	if _, err := h.svc.Send(h.ctx, messages.SendInput{
		ChatType: "group", ChatID: h.group, Text: "взлом", ThreadRootID: &root.ID,
	}, h.mallory); !errors.Is(err, messages.ErrNotFound) {
		t.Fatalf("foreign thread post: want ErrNotFound, got %v", err)
	}
}

// An expired (disappearing) reply must decrement its root's counter when
// the reaper hard-deletes it (stage J × stage K interaction).
func TestReaperDecrementsThreadCounter(t *testing.T) {
	h := threadSetup(t)
	root := h.send(t, "корень", nil)
	h.send(t, "вечный ответ", &root.ID)
	doomed := h.send(t, "исчезающий ответ", &root.ID)

	if _, err := h.pool.Exec(h.ctx, `UPDATE messages SET expires_at = now() - interval '1 second' WHERE id = $1`, doomed.ID); err != nil {
		t.Fatalf("force expire: %v", err)
	}

	rec := audit.NewPostgresRecorder(slog.New(slog.NewTextHandler(io.Discard, nil)))
	reaper := disappear.NewService(h.pool, disappear.NewPostgresRepository(),
		func(_ context.Context, _ string, _ uuid.UUID, _ uuid.UUID, _ int) error { return nil }, rec)
	if n, err := reaper.ProcessExpired(h.ctx, time.Now(), 100); err != nil || n != 1 {
		t.Fatalf("reap: n=%d, %v", n, err)
	}

	page, err := h.svc.List(h.ctx, "group", h.group, "", 50, h.alice)
	if err != nil || len(page.Items) != 1 {
		t.Fatalf("list after reap: %v, %v", page, err)
	}
	if page.Items[0].ThreadReplyCount != 1 {
		t.Fatalf("root counter after reap = %d, want 1", page.Items[0].ThreadReplyCount)
	}
	thread, _ := h.svc.ListThread(h.ctx, root.ID, "", 50, h.alice)
	if len(thread.Items) != 1 {
		t.Fatalf("thread after reap = %d items, want 1", len(thread.Items))
	}
}
