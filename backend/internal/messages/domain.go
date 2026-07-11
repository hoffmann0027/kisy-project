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
	// ErrForwardBroadens rejects a forward whose target audience is broader
	// than the source's — moving content "up" the clearance hierarchy.
	ErrForwardBroadens = errors.New("messages: cannot forward to a broader audience")
	// ErrForwardEncrypted marks an E2EE source the server cannot forward
	// itself; the client must decrypt and re-send (docs/e2ee-design.md).
	ErrForwardEncrypted = errors.New("messages: encrypted messages are forwarded client-side")
)

// Chat types.
const (
	ChatPrivate = "private"
	ChatGroup   = "group"
)

// MaxTextLength bounds a single message body.
const MaxTextLength = 8000

// MaxCiphertextBytes bounds an E2EE message body (mirrors the DB CHECK).
const MaxCiphertextBytes = 65536

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

	// E2EE body (docs/e2ee-design.md §6.1): MLS ciphertext the server cannot
	// read. A live message carries either Text or Ciphertext, never both.
	// Alg versions the encryption scheme, Epoch is the MLS epoch, ContentKind
	// says text/attachment/system without revealing the content itself.
	Ciphertext  []byte
	Alg         *int16
	Epoch       *int64
	ContentKind *int16

	// Forwarding (stage D). The source message id is kept for audit only and
	// never exposed; the sender is a snapshot (id + name at forward time) so
	// the forwarded bubble reveals nothing about the source chat.
	ForwardedFromMessageID  *uuid.UUID
	ForwardedFromSenderID   *uuid.UUID
	ForwardedFromSenderName *string

	// Scheduled sending (stage I): the scheduled_messages row this message
	// was born from. Metadata only — it lets the sender's client re-key its
	// locally cached E2EE plaintext (sched/<scheduledId> → msg/<messageId>).
	ScheduledMessageID *uuid.UUID
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
	// ReadCount/ReadTotal are set only for the actor's own group messages:
	// how many recipients have read it out of the total. Nil otherwise.
	ReadCount *int `json:"readCount"`
	ReadTotal *int `json:"readTotal"`

	// E2EE body: MLS ciphertext as base64 ([]byte's default JSON encoding).
	// Present only on encrypted messages; Text is nil for those.
	Ciphertext  []byte `json:"ciphertext,omitempty"`
	Alg         *int16 `json:"alg,omitempty"`
	Epoch       *int64 `json:"epoch,omitempty"`
	ContentKind *int16 `json:"contentKind,omitempty"`

	// Forwarding attribution: who originally wrote this, as a snapshot. The
	// source chat/message is never exposed, so no cross-clearance leak.
	ForwardedFrom *ForwardedFrom `json:"forwardedFrom,omitempty"`

	// Scheduled origin (stage I) — present only on messages sent by the
	// scheduler, so the sender's client can restore its plaintext cache.
	ScheduledID *uuid.UUID `json:"scheduledId,omitempty"`
}

// ForwardedFrom is the "Переслано от …" attribution shown on a forwarded
// message. It carries only the original author snapshot.
type ForwardedFrom struct {
	SenderID   uuid.UUID `json:"senderId"`
	SenderName string    `json:"senderName"`
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
		dto.Ciphertext = m.Ciphertext
		dto.Alg = m.Alg
		dto.Epoch = m.Epoch
		dto.ContentKind = m.ContentKind
		dto.ScheduledID = m.ScheduledMessageID
		if m.ForwardedFromSenderID != nil {
			name := ""
			if m.ForwardedFromSenderName != nil {
				name = *m.ForwardedFromSenderName
			}
			dto.ForwardedFrom = &ForwardedFrom{SenderID: *m.ForwardedFromSenderID, SenderName: name}
		}
	}
	return dto
}
