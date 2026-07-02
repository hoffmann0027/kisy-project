package messages

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestToDTOHidesDeletedText(t *testing.T) {
	text := "secret content"
	deletedAt := time.Now().UTC()
	m := &Message{
		ID:        uuid.New(),
		ChatType:  ChatPrivate,
		ChatID:    uuid.New(),
		SenderID:  uuid.New(),
		Text:      &text,
		IsDeleted: true,
		DeletedAt: &deletedAt,
	}

	dto := m.ToDTO()
	if dto.Text != nil {
		t.Fatalf("deleted message must not expose text, got %q", *dto.Text)
	}
	if !dto.IsDeleted || dto.DeletedAt == nil {
		t.Fatal("tombstone flags must be preserved")
	}
	// The slot in the timeline remains: id and timestamps still present.
	if dto.ID != m.ID {
		t.Fatal("deleted message must keep its id")
	}
}

func TestToDTOKeepsLiveText(t *testing.T) {
	text := "hello world"
	m := &Message{
		ID:       uuid.New(),
		ChatType: ChatGroup,
		ChatID:   uuid.New(),
		SenderID: uuid.New(),
		Text:     &text,
	}

	dto := m.ToDTO()
	if dto.Text == nil || *dto.Text != text {
		t.Fatal("live message must expose its text")
	}
	// Contract fields are always non-nil arrays, never null.
	if dto.Attachments == nil || dto.Reactions == nil || dto.Mentions == nil {
		t.Fatal("attachments/reactions/mentions must serialize as arrays, not null")
	}
}
