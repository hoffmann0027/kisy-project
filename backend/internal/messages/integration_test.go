//go:build integration

package messages_test

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"

	"github.com/google/uuid"

	"kisy-backend/internal/audit"
	"kisy-backend/internal/chats"
	"kisy-backend/internal/messages"
	"kisy-backend/internal/platform/testdb"
	"kisy-backend/internal/reactions"
	"kisy-backend/internal/readstate"
)

type harness struct {
	msgs  *messages.Service
	react *reactions.Service
	reads *readstate.Service
	chat  *chats.PrivateChat
	a, b  uuid.UUID
	ctx   context.Context
}

func setup(t *testing.T) harness {
	pool := testdb.New(t)
	ctx := context.Background()
	rec := audit.NewPostgresRecorder(slog.New(slog.NewTextHandler(io.Discard, nil)))

	a := testdb.SeedUser(t, pool, "alice", 3)
	b := testdb.SeedUser(t, pool, "bob", 8)
	levels := map[uuid.UUID]int{a: 3, b: 8}
	lookup := func(_ context.Context, id uuid.UUID) (int, bool) {
		lvl, ok := levels[id]
		return lvl, ok
	}

	chatsSvc := chats.NewService(pool, chats.NewPostgresRepository(), lookup)
	chat, err := chatsSvc.OpenPrivateChat(ctx, b, chats.ActorMeta{UserID: a, RoleLevel: 3})
	if err != nil {
		t.Fatalf("open chat: %v", err)
	}

	authz := messages.Authorizer{
		Private: func(ctx context.Context, chatID, actorID uuid.UUID) error {
			ok, err := chatsSvc.IsParticipant(ctx, chatID, actorID)
			if err != nil {
				return err
			}
			if !ok {
				return messages.ErrNotFound
			}
			return nil
		},
		Group: func(context.Context, uuid.UUID, uuid.UUID, int) error { return messages.ErrNotFound },
	}
	msgs := messages.NewService(pool, messages.NewPostgresRepository(), rec, authz)
	react := reactions.NewService(pool, reactions.NewPostgresRepository(), msgs)
	msgs.SetReactionLoader(react.Loader)

	chatAuthz := func(ctx context.Context, _ string, chatID, actorID uuid.UUID, _ int) error {
		ok, err := chatsSvc.IsParticipant(ctx, chatID, actorID)
		if err != nil {
			return err
		}
		if !ok {
			return errors.New("not a participant")
		}
		return nil
	}
	reads := readstate.NewService(pool, readstate.NewPostgresRepository(), chatAuthz)

	return harness{msgs: msgs, react: react, reads: reads, chat: chat, a: a, b: b, ctx: ctx}
}

func actor(id uuid.UUID, level int) messages.ActorMeta {
	return messages.ActorMeta{UserID: id, RoleLevel: level}
}

// encrypted builds an E2EE body; the bytes stand in for MLS ciphertext, which
// the server never reads.
func encrypted(chatType string, chatID uuid.UUID, label string) messages.SendInput {
	alg, epoch := int16(1), int64(1)
	return messages.SendInput{ChatType: chatType, ChatID: chatID, Ciphertext: []byte("ciphertext:" + label), Alg: &alg, Epoch: &epoch}
}

func TestMessageLifecycle(t *testing.T) {
	h := setup(t)
	cid := h.chat.ID

	// Private chats carry only ciphertext (audit A-10).
	m1, err := h.msgs.Send(h.ctx, encrypted("private", cid, "hello"), actor(h.a, 3))
	if err != nil {
		t.Fatalf("send: %v", err)
	}
	m2, err := h.msgs.Send(h.ctx, encrypted("private", cid, "hi"), actor(h.b, 8))
	if err != nil {
		t.Fatalf("send b: %v", err)
	}

	// Newest-first ordering.
	page, err := h.msgs.List(h.ctx, "private", cid, "", 50, actor(h.a, 3))
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(page.Items) != 2 || page.Items[0].ID != m2.ID {
		t.Fatalf("expected 2 messages newest-first, got %+v", page.Items)
	}

	// Delete policy: bob cannot delete alice's message; alice can.
	if err := h.msgs.Delete(h.ctx, m1.ID, actor(h.b, 8)); !errors.Is(err, messages.ErrForbidden) {
		t.Fatalf("bob delete alice msg: got %v, want ErrForbidden", err)
	}
	if err := h.msgs.Delete(h.ctx, m1.ID, actor(h.a, 3)); err != nil {
		t.Fatalf("alice delete own: %v", err)
	}
	page, _ = h.msgs.List(h.ctx, "private", cid, "", 50, actor(h.a, 3))
	var tomb *messages.DTO
	for i := range page.Items {
		if page.Items[i].ID == m1.ID {
			tomb = &page.Items[i]
		}
	}
	if tomb == nil || !tomb.IsDeleted || tomb.Text != nil || len(tomb.Ciphertext) != 0 {
		t.Fatalf("deleted message should be a content-less tombstone: %+v", tomb)
	}
}

func TestReactionsPerViewer(t *testing.T) {
	h := setup(t)
	cid := h.chat.ID
	m, _ := h.msgs.Send(h.ctx, encrypted("private", cid, "react to me"), actor(h.a, 3))

	if err := h.react.Add(h.ctx, m.ID, "👍", reactions.Actor{UserID: h.b, RoleLevel: 8}); err != nil {
		t.Fatalf("add reaction: %v", err)
	}

	// Bob sees reacted=true, Alice sees reacted=false, both count 1.
	forB, _ := h.msgs.List(h.ctx, "private", cid, "", 50, actor(h.b, 8))
	rb := reactionOf(forB.Items, m.ID)
	if rb == nil || rb.Count != 1 || !rb.Reacted {
		t.Fatalf("bob reaction view wrong: %+v", rb)
	}
	forA, _ := h.msgs.List(h.ctx, "private", cid, "", 50, actor(h.a, 3))
	ra := reactionOf(forA.Items, m.ID)
	if ra == nil || ra.Count != 1 || ra.Reacted {
		t.Fatalf("alice reaction view wrong: %+v", ra)
	}
}

func TestUnreadCounters(t *testing.T) {
	h := setup(t)
	cid := h.chat.ID
	mb, _ := h.msgs.Send(h.ctx, encrypted("private", cid, "for alice"), actor(h.b, 8))

	// Alice has one unread (bob's message).
	counts, err := h.reads.UnreadForPrivateChats(h.ctx, h.a, []uuid.UUID{cid})
	if err != nil {
		t.Fatal(err)
	}
	if counts[cid] != 1 {
		t.Fatalf("alice unread = %d, want 1", counts[cid])
	}

	// After marking read, zero.
	if err := h.reads.MarkRead(h.ctx, "private", cid, mb.ID, readstate.Actor{UserID: h.a, RoleLevel: 3}); err != nil {
		t.Fatal(err)
	}
	counts, _ = h.reads.UnreadForPrivateChats(h.ctx, h.a, []uuid.UUID{cid})
	if counts[cid] != 0 {
		t.Fatalf("alice unread after read = %d, want 0", counts[cid])
	}
}

func reactionOf(items []messages.DTO, id uuid.UUID) *messages.ReactionSummary {
	for i := range items {
		if items[i].ID == id && len(items[i].Reactions) > 0 {
			return &items[i].Reactions[0]
		}
	}
	return nil
}
