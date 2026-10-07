//go:build integration

package messages_test

import (
	"errors"
	"testing"

	"github.com/google/uuid"

	"kisy-backend/internal/messages"
	"kisy-backend/internal/platform/testdb"
)

// Audit A-34: "forwarded from" was whatever the client wrote — any id, any
// name, in any chat ("forwardedFromSenderName":"CEO" → «Переслано от CEO»,
// and a 900 000-character name was stored as well). Now the client names the
// source message and the server reads the attribution from it.

func TestForwardAttributionComesFromTheSource(t *testing.T) {
	h := fwdSetup(t)
	encrypted := func(chatID, sender uuid.UUID) uuid.UUID {
		var id uuid.UUID
		if err := h.pool.QueryRow(h.ctx, `
			INSERT INTO messages (chat_type, chat_id, sender_id, ciphertext, alg, epoch)
			VALUES ('private', $1, $2, decode('00', 'hex'), 1, 1) RETURNING id`, chatID, sender).Scan(&id); err != nil {
			t.Fatal(err)
		}
		return id
	}
	send := func(source uuid.UUID) (messages.DTO, error) {
		return h.msgs.Send(h.ctx, messages.SendInput{
			ChatType: messages.ChatGroup, ChatID: h.wideGroup, Text: "расшифрованный текст",
			ForwardedFromMessageID: &source,
		}, fwdActor(h.a, 3))
	}

	// Bob wrote it in his chat with alice: the forward says bob, by the
	// server's name for him.
	fromBob := encrypted(h.privChat, h.b)
	dto, err := send(fromBob)
	if err != nil {
		t.Fatalf("forwarding a message alice can read: %v", err)
	}
	if dto.ForwardedFrom == nil || dto.ForwardedFrom.SenderID != h.b || dto.ForwardedFrom.SenderName != "bob" {
		t.Fatalf("attribution %+v, want bob", dto.ForwardedFrom)
	}

	// A message from a chat alice is not in cannot lend its author.
	carol := testdb.SeedUser(t, h.pool, "carol", 5)
	var foreign uuid.UUID
	if err := h.pool.QueryRow(h.ctx, `INSERT INTO private_chats (user_a_id, user_b_id, initiated_by) VALUES ($1, $2, $1) RETURNING id`, h.b, carol).Scan(&foreign); err != nil {
		t.Fatal(err)
	}
	if _, err := send(encrypted(foreign, carol)); !errors.Is(err, messages.ErrNotFound) {
		t.Fatalf("attribution borrowed from a chat alice cannot read: %v", err)
	}

	// A plaintext source is the server's to forward, under the audience rule.
	plain, err := h.msgs.Send(h.ctx, messages.SendInput{
		ChatType: messages.ChatGroup, ChatID: h.narrowGrp, Text: "только для узкого круга",
	}, fwdActor(h.a, 3))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := send(plain.ID); !errors.Is(err, messages.ErrForbidden) {
		t.Fatalf("a plaintext source went around Forward: %v", err)
	}
}
