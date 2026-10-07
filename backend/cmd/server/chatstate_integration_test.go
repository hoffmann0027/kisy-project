//go:build integration

package main

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	"kisy-backend/internal/notifprefs"
	"kisy-backend/internal/platform/testdb"
	"kisy-backend/internal/readstate"
)

// Audit A-37: muting accepted any chat id, and a read receipt accepted any
// message id — one from somebody else's chat included — which was then
// broadcast to this chat's members as the reader's position.

func TestMutingIsOnlyForChatsYouAreIn(t *testing.T) {
	pool := testdb.New(t)
	ctx := context.Background()
	alice := testdb.SeedUser(t, pool, "alice_"+uuid.NewString()[:6], 5)
	bob := testdb.SeedUser(t, pool, "bob_"+uuid.NewString()[:6], 5)
	outsider := testdb.SeedUser(t, pool, "out_"+uuid.NewString()[:6], 5)
	var chat uuid.UUID
	if err := pool.QueryRow(ctx, `INSERT INTO private_chats (user_a_id, user_b_id, initiated_by) VALUES ($1, $2, $1) RETURNING id`, alice, bob).Scan(&chat); err != nil {
		t.Fatal(err)
	}

	svc := notifprefs.NewService(pool, notifprefs.NewPostgresRepository())
	svc.SetChatMember(chatMember(pool))

	if err := svc.Mute(ctx, outsider, "private", chat, nil); !errors.Is(err, notifprefs.ErrNotFound) {
		t.Fatalf("an outsider muted somebody else's chat: %v", err)
	}
	if err := svc.Mute(ctx, alice, "private", chat, nil); err != nil {
		t.Fatalf("a participant could not mute their own chat: %v", err)
	}
}

func TestReadPositionMustBeAMessageOfThatChat(t *testing.T) {
	pool := testdb.New(t)
	ctx := context.Background()
	alice := testdb.SeedUser(t, pool, "alice_"+uuid.NewString()[:6], 5)
	bob := testdb.SeedUser(t, pool, "bob_"+uuid.NewString()[:6], 5)
	carol := testdb.SeedUser(t, pool, "carol_"+uuid.NewString()[:6], 5)

	chat := func(a, b uuid.UUID) uuid.UUID {
		var id uuid.UUID
		if err := pool.QueryRow(ctx, `INSERT INTO private_chats (user_a_id, user_b_id, initiated_by) VALUES ($1, $2, $1) RETURNING id`, a, b).Scan(&id); err != nil {
			t.Fatal(err)
		}
		return id
	}
	message := func(chatID, sender uuid.UUID) uuid.UUID {
		var id uuid.UUID
		if err := pool.QueryRow(ctx, `
			INSERT INTO messages (chat_type, chat_id, sender_id, ciphertext, alg, epoch)
			VALUES ('private', $1, $2, decode('00', 'hex'), 1, 1) RETURNING id`, chatID, sender).Scan(&id); err != nil {
			t.Fatal(err)
		}
		return id
	}
	ours := chat(alice, bob)
	theirs := chat(bob, carol)
	inOurs := message(ours, bob)
	inTheirs := message(theirs, carol)

	svc := readstate.NewService(pool, readstate.NewPostgresRepository(),
		func(context.Context, string, uuid.UUID, uuid.UUID, int) error { return nil })

	if svc.PersistRead(ctx, alice, "private", ours, inTheirs) {
		t.Fatal("a message from another chat was stored as alice's read position")
	}
	if !svc.PersistRead(ctx, alice, "private", ours, inOurs) {
		t.Fatal("a message of the chat itself was refused")
	}
	var stored uuid.UUID
	if err := pool.QueryRow(ctx, `SELECT last_read_message_id FROM chat_read_state WHERE user_id = $1 AND chat_id = $2`, alice, ours).Scan(&stored); err != nil {
		t.Fatal(err)
	}
	if stored != inOurs {
		t.Fatalf("stored position %v, want %v", stored, inOurs)
	}
}
