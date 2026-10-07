//go:build integration

package messages_test

import (
	"errors"
	"testing"

	"github.com/google/uuid"

	"kisy-backend/internal/groups"
	"kisy-backend/internal/messages"
)

// Audit A-14: a plain reader of an editors-only channel could not send there,
// but could write into it through three side doors — a forward, a pin, and
// (in internal/disappear) the chat's disappearing timer. All three are writing.
func TestReaderOfAnEditorsOnlyGroupCannotWriteThroughSideDoors(t *testing.T) {
	h := fwdSetup(t)

	// alice's broad group becomes editors-only; bob (level 8) joins as a
	// plain member.
	if _, err := h.pool.Exec(h.ctx, `UPDATE groups SET post_policy = 'editors' WHERE id = $1`, h.wideGroup); err != nil {
		t.Fatal(err)
	}
	if _, err := h.pool.Exec(h.ctx, `INSERT INTO group_members (group_id, user_id) VALUES ($1, $2)`, h.wideGroup, h.b); err != nil {
		t.Fatal(err)
	}

	// The front door is shut, as it always was.
	if _, err := h.msgs.Send(h.ctx, messages.SendInput{ChatType: "group", ChatID: h.wideGroup, Text: "можно?"}, fwdActor(h.b, 8)); !errors.Is(err, messages.ErrForbidden) {
		t.Fatalf("a reader sent into an editors-only group: %v", err)
	}

	// bob has a group of his own to forward from.
	own, err := h.groups.Create(h.ctx, groups.CreateInput{Name: "bob's", MinRoleLevel: 8}, groups.ActorMeta{UserID: h.b, RoleLevel: 8})
	if err != nil {
		t.Fatal(err)
	}
	src, err := h.msgs.Send(h.ctx, messages.SendInput{ChatType: "group", ChatID: own.ID, Text: "через пересылку"}, fwdActor(h.b, 8))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := h.msgs.Forward(h.ctx, messages.ForwardInput{
		SourceMessageIDs: []uuid.UUID{src.ID}, TargetChatType: "group", TargetChatID: h.wideGroup,
	}, fwdActor(h.b, 8)); !errors.Is(err, messages.ErrForbidden) {
		t.Fatalf("a reader forwarded into an editors-only group: %v", err)
	}

	// An editor's post cannot be pinned or unpinned by a reader either.
	post, err := h.msgs.Send(h.ctx, messages.SendInput{ChatType: "group", ChatID: h.wideGroup, Text: "от редакции"}, fwdActor(h.a, 3))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := h.msgs.SetPinned(h.ctx, post.ID, true, fwdActor(h.b, 8)); !errors.Is(err, messages.ErrForbidden) {
		t.Fatalf("a reader pinned a message in an editors-only group: %v", err)
	}
	// The owner still can.
	if _, err := h.msgs.SetPinned(h.ctx, post.ID, true, fwdActor(h.a, 3)); err != nil {
		t.Fatalf("the owner could not pin: %v", err)
	}
}

// Audit A-15: "no threshold" is the broadest audience there is — it includes
// accounts outside the hierarchy — but compared as a raw 0 it read as the
// narrowest, so a forward from a level-3 group into an open group went
// through.
func TestForwardIntoAnOpenGroupBroadensTheAudience(t *testing.T) {
	h := fwdSetup(t)

	open, err := h.groups.Create(h.ctx, groups.CreateInput{Name: "open to all"}, groups.ActorMeta{UserID: h.a, RoleLevel: 3})
	if err != nil {
		t.Fatal(err)
	}
	src, err := h.msgs.Send(h.ctx, messages.SendInput{ChatType: "group", ChatID: h.narrowGrp, Text: "для узкого круга"}, fwdActor(h.a, 3))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := h.msgs.Forward(h.ctx, messages.ForwardInput{
		SourceMessageIDs: []uuid.UUID{src.ID}, TargetChatType: "group", TargetChatID: open.ID,
	}, fwdActor(h.a, 3)); !errors.Is(err, messages.ErrForwardBroadens) {
		t.Fatalf("a level-3 message was forwarded into an open group: %v", err)
	}
}

func TestPrivateChatWithABasicAccountIsTheBroadestAudience(t *testing.T) {
	// A level-3 person talking to an account outside the hierarchy: whatever
	// lands there, the basic account reads.
	if got := messages.PrivateChatAudience([]int{3, 0}); got != 0 {
		t.Fatalf("audience = %d, want NoLevel (0): the basic participant was ignored", got)
	}
	if got := messages.PrivateChatAudience([]int{3, 8}); got != 8 {
		t.Fatalf("audience = %d, want 8 (the weaker participant)", got)
	}
}
