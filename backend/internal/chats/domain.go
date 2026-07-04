// Package chats owns one-to-one private conversations and the unified
// "list my chats" view.
package chats

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

var (
	ErrNotFound       = errors.New("chats: not found")
	ErrNotParticipant = errors.New("chats: actor is not a participant")
	ErrCannotInitiate = errors.New("chats: clearance does not permit initiating this chat")
	ErrSelfChat       = errors.New("chats: cannot open a chat with yourself")
)

// PrivateChat mirrors the private_chats table.
type PrivateChat struct {
	ID          uuid.UUID
	UserAID     uuid.UUID
	UserBID     uuid.UUID
	InitiatedBy uuid.UUID
	CreatedAt   time.Time
}

// Other returns the participant that is not self.
func (c *PrivateChat) Other(self uuid.UUID) uuid.UUID {
	if c.UserAID == self {
		return c.UserBID
	}
	return c.UserAID
}

// HasParticipant reports whether userID is one of the two participants.
func (c *PrivateChat) HasParticipant(userID uuid.UUID) bool {
	return c.UserAID == userID || c.UserBID == userID
}

// DTO is the API representation of a private chat. OtherUser carries the
// counterpart's public profile so clients can render the chat without a
// separate lookup; it is nil only if that account was removed.
type DTO struct {
	ID          uuid.UUID `json:"id"`
	Type        string    `json:"type"` // always "private" here
	OtherUserID uuid.UUID `json:"otherUserId"`
	OtherUser   any       `json:"otherUser"`
	UnreadCount int       `json:"unreadCount"`
	// OtherLastReadAt is when the counterpart last read this chat, powering
	// read receipts on the actor's own messages. Nil if they never read it.
	OtherLastReadAt *time.Time `json:"otherLastReadAt"`
	CreatedAt       time.Time  `json:"createdAt"`
}

func (c *PrivateChat) ToDTO(self uuid.UUID) DTO {
	return DTO{
		ID:          c.ID,
		Type:        "private",
		OtherUserID: c.Other(self),
		CreatedAt:   c.CreatedAt,
	}
}
