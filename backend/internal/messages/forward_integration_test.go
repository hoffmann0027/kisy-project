//go:build integration

package messages_test

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"kisy-backend/internal/audit"
	"kisy-backend/internal/chats"
	"kisy-backend/internal/groups"
	"kisy-backend/internal/messages"
	"kisy-backend/internal/platform/testdb"
)

type fwdHarness struct {
	msgs      *messages.Service
	pool      *pgxpool.Pool
	privChat  uuid.UUID // alice(3) <-> bob(8): breadth 8
	wideGroup uuid.UUID // min_role_level 8 (broad)
	narrowGrp uuid.UUID // min_role_level 3 (restricted)
	a, b      uuid.UUID
	ctx       context.Context
}

func fwdSetup(t *testing.T) fwdHarness {
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

	groupsSvc := groups.NewService(pool, groups.NewPostgresRepository(), rec)
	// alice (level 3) creates a restricted group (min 3) and a broad group
	// (min 8). Both are created by alice so she is a member of each.
	wide, err := groupsSvc.Create(ctx, groups.CreateInput{Name: "wide", MinRoleLevel: 8}, groups.ActorMeta{UserID: a, RoleLevel: 3})
	if err != nil {
		t.Fatalf("create wide group: %v", err)
	}
	narrow, err := groupsSvc.Create(ctx, groups.CreateInput{Name: "narrow", MinRoleLevel: 3}, groups.ActorMeta{UserID: a, RoleLevel: 3})
	if err != nil {
		t.Fatalf("create narrow group: %v", err)
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
		Group: func(ctx context.Context, groupID, actorID uuid.UUID, actorLevel int) error {
			err := groupsSvc.EnsureMember(ctx, groupID, groups.ActorMeta{UserID: actorID, RoleLevel: actorLevel})
			switch {
			case errors.Is(err, groups.ErrNotFound):
				return messages.ErrNotFound
			case errors.Is(err, groups.ErrNotMember):
				return messages.ErrForbidden
			default:
				return err
			}
		},
	}
	msgs := messages.NewService(pool, messages.NewPostgresRepository(), rec, authz)
	msgs.SetForwarding(
		messages.ClearanceResolver{
			Private: func(ctx context.Context, chatID uuid.UUID) (int, error) {
				ids, err := chatsSvc.ParticipantIDs(ctx, chatID)
				if err != nil {
					return 0, err
				}
				breadth := 1
				for _, id := range ids {
					if lvl, ok := lookup(ctx, id); ok && lvl > breadth {
						breadth = lvl
					}
				}
				return breadth, nil
			},
			Group: func(ctx context.Context, groupID uuid.UUID) (int, error) {
				return groupsSvc.ClearanceLevel(ctx, groupID)
			},
		},
		func(ctx context.Context, userID uuid.UUID) (string, bool) {
			names := map[uuid.UUID]string{a: "alice", b: "bob"}
			n, ok := names[userID]
			return n, ok
		},
		nil, // attachment copier not needed for these tests
	)

	return fwdHarness{
		msgs: msgs, pool: pool, privChat: chat.ID,
		wideGroup: wide.ID, narrowGrp: narrow.ID, a: a, b: b, ctx: ctx,
	}
}

func fwdActor(id uuid.UUID, level int) messages.ActorMeta {
	return messages.ActorMeta{UserID: id, RoleLevel: level}
}

func TestForwardHappyPathAndAttribution(t *testing.T) {
	h := fwdSetup(t)

	// bob writes in the private chat; alice forwards it to the narrow group.
	src, err := h.msgs.Send(h.ctx, messages.SendInput{ChatType: "private", ChatID: h.privChat, Text: "секрет"}, fwdActor(h.b, 8))
	if err != nil {
		t.Fatalf("send source: %v", err)
	}
	out, err := h.msgs.Forward(h.ctx, messages.ForwardInput{
		SourceMessageIDs: []uuid.UUID{src.ID},
		TargetChatType:   "group",
		TargetChatID:     h.narrowGrp,
	}, fwdActor(h.a, 3))
	if err != nil {
		t.Fatalf("forward: %v", err)
	}
	if len(out) != 1 {
		t.Fatalf("expected 1 forwarded message, got %d", len(out))
	}
	fwd := out[0]
	if fwd.Text == nil || *fwd.Text != "секрет" {
		t.Fatalf("forwarded text mismatch: %+v", fwd.Text)
	}
	if fwd.ForwardedFrom == nil || fwd.ForwardedFrom.SenderID != h.b || fwd.ForwardedFrom.SenderName != "bob" {
		t.Fatalf("attribution snapshot wrong: %+v", fwd.ForwardedFrom)
	}
	// The forward is a fresh message authored by the forwarder.
	if fwd.SenderID != h.a {
		t.Fatalf("forwarded message must be authored by the forwarder")
	}
}

func TestForwardCannotBroadenAudience(t *testing.T) {
	h := fwdSetup(t)

	// alice posts in the NARROW group (min 3, restricted) and tries to
	// forward it to the WIDE group (min 8, broad) — this broadens the
	// audience and must be rejected.
	src, err := h.msgs.Send(h.ctx, messages.SendInput{ChatType: "group", ChatID: h.narrowGrp, Text: "для узкого круга"}, fwdActor(h.a, 3))
	if err != nil {
		t.Fatalf("send in narrow group: %v", err)
	}
	if _, err := h.msgs.Forward(h.ctx, messages.ForwardInput{
		SourceMessageIDs: []uuid.UUID{src.ID},
		TargetChatType:   "group",
		TargetChatID:     h.wideGroup,
	}, fwdActor(h.a, 3)); !errors.Is(err, messages.ErrForwardBroadens) {
		t.Fatalf("broaden: want ErrForwardBroadens, got %v", err)
	}

	// The reverse (wide -> narrow, narrowing) is allowed.
	src2, err := h.msgs.Send(h.ctx, messages.SendInput{ChatType: "group", ChatID: h.wideGroup, Text: "широкая аудитория"}, fwdActor(h.a, 3))
	if err != nil {
		t.Fatalf("send in wide group: %v", err)
	}
	if _, err := h.msgs.Forward(h.ctx, messages.ForwardInput{
		SourceMessageIDs: []uuid.UUID{src2.ID},
		TargetChatType:   "group",
		TargetChatID:     h.narrowGrp,
	}, fwdActor(h.a, 3)); err != nil {
		t.Fatalf("narrowing forward should be allowed: %v", err)
	}
}

func TestForwardInaccessibleSourceMasked(t *testing.T) {
	h := fwdSetup(t)

	// A message in a chat the actor cannot access must read as not-found,
	// never revealing its existence. bob (level 8) cannot see the narrow
	// group (min 3).
	src, err := h.msgs.Send(h.ctx, messages.SendInput{ChatType: "group", ChatID: h.narrowGrp, Text: "скрытое"}, fwdActor(h.a, 3))
	if err != nil {
		t.Fatalf("send: %v", err)
	}
	if _, err := h.msgs.Forward(h.ctx, messages.ForwardInput{
		SourceMessageIDs: []uuid.UUID{src.ID},
		TargetChatType:   "private",
		TargetChatID:     h.privChat,
	}, fwdActor(h.b, 8)); !errors.Is(err, messages.ErrNotFound) {
		t.Fatalf("inaccessible source: want ErrNotFound, got %v", err)
	}
}

func TestForwardEncryptedRejected(t *testing.T) {
	h := fwdSetup(t)

	alg := int16(1)
	src, err := h.msgs.Send(h.ctx, messages.SendInput{
		ChatType: "private", ChatID: h.privChat, Ciphertext: []byte{1, 2, 3, 4}, Alg: &alg,
	}, fwdActor(h.a, 3))
	if err != nil {
		t.Fatalf("send e2ee: %v", err)
	}
	if _, err := h.msgs.Forward(h.ctx, messages.ForwardInput{
		SourceMessageIDs: []uuid.UUID{src.ID},
		TargetChatType:   "group",
		TargetChatID:     h.narrowGrp,
	}, fwdActor(h.a, 3)); !errors.Is(err, messages.ErrForwardEncrypted) {
		t.Fatalf("e2ee source: want ErrForwardEncrypted, got %v", err)
	}
}
