//go:build integration

package messages_test

import (
	"testing"

	"github.com/google/uuid"

	"kisy-backend/internal/messages"
)

// In a group the bubble has to say who wrote it. The name travels with the
// message — in the live event of a new one, in the history, in a forward and
// among the pinned — so the app never has to know every member to show it.
func TestAGroupMessageSaysWhoWroteIt(t *testing.T) {
	h := fwdSetup(t)

	sent, err := h.msgs.Send(h.ctx, messages.SendInput{ChatType: "group", ChatID: h.wideGroup, Text: "привет"}, fwdActor(h.a, 3))
	if err != nil {
		t.Fatalf("send: %v", err)
	}
	if sent.SenderName != "alice" {
		t.Fatalf("new message: senderName = %q, want alice", sent.SenderName)
	}

	page, err := h.msgs.List(h.ctx, "group", h.wideGroup, "", 50, fwdActor(h.a, 3))
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(page.Items) != 1 || page.Items[0].SenderName != "alice" {
		t.Fatalf("history: %+v", page.Items)
	}

	out, err := h.msgs.Forward(h.ctx, messages.ForwardInput{
		SourceMessageIDs: []uuid.UUID{sent.ID}, TargetChatType: "group", TargetChatID: h.narrowGrp,
	}, fwdActor(h.a, 3))
	if err != nil || len(out) != 1 || out[0].SenderName != "alice" {
		t.Fatalf("forward: %v %+v", err, out)
	}

	if _, err := h.msgs.SetPinned(h.ctx, sent.ID, true, fwdActor(h.a, 3)); err != nil {
		t.Fatalf("pin: %v", err)
	}
	pinned, err := h.msgs.ListPinned(h.ctx, "group", h.wideGroup, fwdActor(h.a, 3))
	if err != nil || len(pinned) != 1 || pinned[0].SenderName != "alice" {
		t.Fatalf("pinned: %v %+v", err, pinned)
	}
}
