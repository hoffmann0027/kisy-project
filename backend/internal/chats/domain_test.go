package chats

import (
	"testing"

	"github.com/google/uuid"
)

func TestOtherAndHasParticipant(t *testing.T) {
	a := uuid.New()
	b := uuid.New()
	stranger := uuid.New()
	chat := &PrivateChat{ID: uuid.New(), UserAID: a, UserBID: b}

	if chat.Other(a) != b {
		t.Error("Other(a) should be b")
	}
	if chat.Other(b) != a {
		t.Error("Other(b) should be a")
	}
	if !chat.HasParticipant(a) || !chat.HasParticipant(b) {
		t.Error("both participants must be recognized")
	}
	if chat.HasParticipant(stranger) {
		t.Error("a non-participant must not be recognized")
	}
}

func TestToDTOUsesOtherParticipant(t *testing.T) {
	self := uuid.New()
	other := uuid.New()
	chat := &PrivateChat{ID: uuid.New(), UserAID: self, UserBID: other}

	dto := chat.ToDTO(self)
	if dto.Type != "private" {
		t.Fatalf("type = %q, want private", dto.Type)
	}
	if dto.OtherUserID != other {
		t.Fatal("DTO must show the other participant relative to self")
	}
}
