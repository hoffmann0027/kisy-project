//go:build integration

package messages_test

import (
	"errors"
	"testing"

	"github.com/google/uuid"

	"kisy-backend/internal/messages"
)

// Audit A-10: the server accepted plain `text` in a private chat — from a
// client whose encryption failed, from an old client, or from a server-side
// forward — and stored it next to the chat's ciphertext. A private chat is
// end-to-end encrypted or it carries no text at all: fail closed, never store
// the clear.

var testCiphertext = []byte("mls-ciphertext-bytes-the-server-cannot-read")

func encryptedBody() (ciphertext []byte, alg *int16, epoch *int64) {
	a, e := int16(1), int64(1)
	return testCiphertext, &a, &e
}

func privateTextCount(t *testing.T, h fwdHarness) int {
	t.Helper()
	var n int
	if err := h.pool.QueryRow(h.ctx,
		`SELECT count(*) FROM messages WHERE chat_type = 'private' AND chat_id = $1 AND text IS NOT NULL`,
		h.privChat).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

func TestPrivateChatRefusesPlaintext(t *testing.T) {
	h := fwdSetup(t)

	_, err := h.msgs.Send(h.ctx, messages.SendInput{ChatType: "private", ChatID: h.privChat, Text: "in the clear"}, fwdActor(h.a, 3))
	if !errors.Is(err, messages.ErrEncryptionRequired) {
		t.Fatalf("plaintext into a private chat: want ErrEncryptionRequired, got %v", err)
	}
	if n := privateTextCount(t, h); n != 0 {
		t.Fatalf("a refused plaintext message was stored anyway (%d rows with text)", n)
	}

	ct, alg, epoch := encryptedBody()
	if _, err := h.msgs.Send(h.ctx, messages.SendInput{ChatType: "private", ChatID: h.privChat, Ciphertext: ct, Alg: alg, Epoch: epoch},
		fwdActor(h.a, 3)); err != nil {
		t.Fatalf("ciphertext into a private chat must still work: %v", err)
	}
	// Groups are not end-to-end encrypted yet (audit A-31): plaintext stays allowed there.
	if _, err := h.msgs.Send(h.ctx, messages.SendInput{ChatType: "group", ChatID: h.wideGroup, Text: "group text"},
		fwdActor(h.a, 3)); err != nil {
		t.Fatalf("plaintext into a group: %v", err)
	}
}

func TestPrivateChatRefusesPlaintextEdit(t *testing.T) {
	h := fwdSetup(t)
	ct, alg, epoch := encryptedBody()
	m, err := h.msgs.Send(h.ctx, messages.SendInput{ChatType: "private", ChatID: h.privChat, Ciphertext: ct, Alg: alg, Epoch: epoch},
		fwdActor(h.a, 3))
	if err != nil {
		t.Fatal(err)
	}
	// An edit carries plaintext; on an encrypted message it would sit beside
	// the ciphertext and be shown instead of it (audit A-35).
	if _, err := h.msgs.Edit(h.ctx, m.ID, "replaced in the clear", fwdActor(h.a, 3)); !errors.Is(err, messages.ErrEncryptionRequired) {
		t.Fatalf("plaintext edit in a private chat: want ErrEncryptionRequired, got %v", err)
	}
	if n := privateTextCount(t, h); n != 0 {
		t.Fatalf("a refused edit stored plaintext (%d rows with text)", n)
	}
}

func TestForwardIntoPrivateChatNeedsClientCiphertext(t *testing.T) {
	h := fwdSetup(t)
	src, err := h.msgs.Send(h.ctx, messages.SendInput{ChatType: "group", ChatID: h.wideGroup, Text: "group secret"}, fwdActor(h.a, 3))
	if err != nil {
		t.Fatal(err)
	}

	// Without ciphertext the server would copy the group's clear text into
	// the private chat.
	_, err = h.msgs.Forward(h.ctx, messages.ForwardInput{
		SourceMessageIDs: []uuid.UUID{src.ID}, TargetChatType: "private", TargetChatID: h.privChat,
	}, fwdActor(h.a, 3))
	if !errors.Is(err, messages.ErrEncryptionRequired) {
		t.Fatalf("forward into a private chat without ciphertext: want ErrEncryptionRequired, got %v", err)
	}
	if n := privateTextCount(t, h); n != 0 {
		t.Fatalf("a refused forward stored plaintext (%d rows with text)", n)
	}

	ct, alg, epoch := encryptedBody()
	out, err := h.msgs.Forward(h.ctx, messages.ForwardInput{
		SourceMessageIDs: []uuid.UUID{src.ID}, TargetChatType: "private", TargetChatID: h.privChat,
		Encrypted: map[uuid.UUID]messages.EncryptedText{src.ID: {Ciphertext: ct, Alg: alg, Epoch: epoch}},
	}, fwdActor(h.a, 3))
	if err != nil {
		t.Fatalf("forward with client ciphertext: %v", err)
	}
	if len(out) != 1 || out[0].Text != nil || out[0].ForwardedFrom == nil {
		t.Fatalf("forwarded message must carry ciphertext and attribution, not text: %+v", out)
	}
	if n := privateTextCount(t, h); n != 0 {
		t.Fatalf("forward stored plaintext in the private chat (%d rows)", n)
	}
}
