// Package messages owns chat messages: creation, paginated retrieval,
// sender-only editing and policy-gated deletion.
package messages

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

var (
	ErrNotFound     = errors.New("messages: not found")
	ErrForbidden    = errors.New("messages: not permitted")
	ErrEmptyContent = errors.New("messages: message has no content")
	ErrBadChatType  = errors.New("messages: unknown chat type")
)

// Chat types.
const (
	ChatPrivate = "private"
	ChatGroup   = "group"
)

// MaxTextLength bounds a single message body.
const MaxTextLength = 8000

// Message mirrors the messages table.
type Message struct {
	ID        uuid.UUID
	ChatType  string
	ChatID    uuid.UUID
	SenderID  uuid.UUID
	Text      *string
	ReplyTo   *uuid.UUID
	IsDeleted bool
	DeletedAt *time.Time
	EditedAt  *time.Time
	PinnedAt  *time.Time
	PinnedBy  *uuid.UUID
	CreatedAt time.Time
}

// ReactionSummary aggregates one emoji on a message: how many users chose
// it and whether the viewer is among them.
type ReactionSummary struct {
	Emoji   string `json:"emoji"`
	Count   int    `json:"count"`
	Reacted bool   `json:"reacted"`
}

// DTO follows the Message Object in docs/spec/09-api-contracts.md.
// Attachments and mentions arrive in later stages but the fields are
// present now so the contract is stable. A deleted message is returned as
// a tombstone: its text is cleared but its slot in the timeline remains.
type DTO struct {
	ID          uuid.UUID         `json:"id"`
	ChatID      uuid.UUID         `json:"chatId"`
	ChatType    string            `json:"chatType"`
	SenderID    uuid.UUID         `json:"senderId"`
	Text        *string           `json:"text"`
	ReplyTo     *uuid.UUID        `json:"replyTo"`
	Attachments []any             `json:"attachments"`
	Reactions   []ReactionSummary `json:"reactions"`
	Mentions    []any             `json:"mentions"`
	IsDeleted   bool              `json:"isDeleted"`
	CreatedAt   time.Time         `json:"createdAt"`
	DeletedAt   *time.Time        `json:"deletedAt"`
	EditedAt    *time.Time        `json:"editedAt"`
	PinnedAt    *time.Time        `json:"pinnedAt"`
}

func (m *Message) ToDTO() DTO {
	dto := DTO{
		ID:          m.ID,
		ChatID:      m.ChatID,
		ChatType:    m.ChatType,
		SenderID:    m.SenderID,
		ReplyTo:     m.ReplyTo,
		Attachments: []any{},
		Reactions:   []ReactionSummary{},
		Mentions:    []any{},
		IsDeleted:   m.IsDeleted,
		CreatedAt:   m.CreatedAt,
		DeletedAt:   m.DeletedAt,
		EditedAt:    m.EditedAt,
		PinnedAt:    m.PinnedAt,
	}
	if !m.IsDeleted {
		dto.Text = m.Text
	}
	return dto
}
