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
	EventPollChanged     = "poll.changed"

	// Voice-call signaling (server→client). Client→server call frames are
	// prefix-routed ("call.*") to the calls package, which owns their names.
	EventCallIncoming = "call.incoming"
	EventCallAnswered = "call.answered"
	EventCallICE      = "call.ice"
	EventCallRejected = "call.rejected"
	EventCallCanceled = "call.canceled"
	EventCallEnded    = "call.ended"
	EventCallBusy     = "call.busy"
	EventCallTimeout  = "call.timeout"

	// E2EE (MLS) handshake delivery: a commit/proposal reached the chat, or
	// a welcome awaits one of the user's devices. Payloads are references —
	// clients fetch the ciphertext frames via /api/v1/e2ee.
	EventE2EEHandshake = "e2ee.handshake"
	EventE2EEWelcome   = "e2ee.welcome"

	EventError = "error"
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

	// E2EE body (base64 MLS ciphertext) — mutually exclusive with text.
	Ciphertext  []byte `json:"ciphertext"`
	Alg         *int16 `json:"alg"`
	Epoch       *int64 `json:"epoch"`
	ContentKind *int16 `json:"contentKind"`
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

// --- call signaling (server→client) payloads ---

type callFrom struct {
	ID          uuid.UUID `json:"id"`
	DisplayName string    `json:"displayName"`
	AvatarURL   *string   `json:"avatarUrl"`
}

type callIncomingData struct {
	CallID uuid.UUID `json:"callId"`
	From   callFrom  `json:"from"`
	ChatID uuid.UUID `json:"chatId"`
	SDP    string    `json:"sdp"`
}

type callAnsweredData struct {
	CallID uuid.UUID `json:"callId"`
	SDP    string    `json:"sdp"`
}

type callICEData struct {
	CallID     uuid.UUID       `json:"callId"`
	FromUserID uuid.UUID       `json:"fromUserId"`
	Candidate  json.RawMessage `json:"candidate"`
}

// callRefData carries just the call id (rejected/canceled/busy/timeout).
type callRefData struct {
	CallID uuid.UUID `json:"callId"`
}

type callEndedData struct {
	CallID uuid.UUID `json:"callId"`
	Reason string    `json:"reason"`
}
