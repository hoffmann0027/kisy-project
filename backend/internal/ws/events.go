// Package ws implements the authenticated WebSocket gateway: presence,
// typing indicators, read receipts and real-time message delivery
// (docs/spec/05-api-websocket.md). Cross-instance fan-out goes through
// Redis pub/sub so any node can reach a client connected to any other.
package ws

import (
	"encoding/json"

	"github.com/google/uuid"
)

// Server→Client event names.
const (
	EventMessageCreated  = "message.created"
	EventMessageUpdated  = "message.updated"
	EventMessageDeleted  = "message.deleted"
	EventMessageRead     = "message.read"
	EventTypingStarted   = "typing.started"
	EventTypingStopped   = "typing.stopped"
	EventUserOnline      = "user.online"
	EventUserOffline     = "user.offline"
	EventUserUpdated     = "user.updated"
	EventReactionAdded   = "reaction.added"
	EventReactionRemoved = "reaction.removed"
	EventNotification    = "notification.created"
	EventBoardChanged    = "board.changed"
	EventGroupChanged    = "group.changed"
	EventRatingChanged   = "rating.changed"
	EventError           = "error"
)

// Client→Server message types.
const (
	TypeMessageSend       = "message.send"
	TypeTypingStart       = "typing.start"
	TypeTypingStop        = "typing.stop"
	TypePresenceSubscribe = "presence.subscribe"
	TypeReadConfirmation  = "read.confirmation"
)

// Inbound is a message received from a client.
type Inbound struct {
	Type string          `json:"type"`
	Data json.RawMessage `json:"data"`
}

// Outbound is an event pushed to a client.
type Outbound struct {
	Event string `json:"event"`
	Data  any    `json:"data"`
}

func encode(event string, data any) []byte {
	raw, _ := json.Marshal(Outbound{Event: event, Data: data})
	return raw
}

// --- inbound payloads ---

type chatRef struct {
	ChatType string    `json:"chatType"`
	ChatID   uuid.UUID `json:"chatId"`
}

type sendPayload struct {
	ChatType string     `json:"chatType"`
	ChatID   uuid.UUID  `json:"chatId"`
	Text     string     `json:"text"`
	ReplyTo  *uuid.UUID `json:"replyTo"`
}

type readPayload struct {
	ChatType  string    `json:"chatType"`
	ChatID    uuid.UUID `json:"chatId"`
	MessageID uuid.UUID `json:"messageId"`
}

type subscribePayload struct {
	UserIDs []uuid.UUID `json:"userIds"`
}

// --- outbound payloads ---

type typingData struct {
	ChatType string    `json:"chatType"`
	ChatID   uuid.UUID `json:"chatId"`
	UserID   uuid.UUID `json:"userId"`
}

type presenceData struct {
	UserID uuid.UUID `json:"userId"`
}

type readData struct {
	ChatType  string    `json:"chatType"`
	ChatID    uuid.UUID `json:"chatId"`
	MessageID uuid.UUID `json:"messageId"`
	UserID    uuid.UUID `json:"userId"`
}

type reactionData struct {
	ChatType  string    `json:"chatType"`
	ChatID    uuid.UUID `json:"chatId"`
	MessageID uuid.UUID `json:"messageId"`
	UserID    uuid.UUID `json:"userId"`
	Emoji     string    `json:"emoji"`
}
