package ws

import (
	"encoding/json"

	"github.com/google/uuid"

	"kisy-backend/internal/messages"
)

// Publisher adapts the hub to the messages.Publisher port so the messages
// service can push created/deleted events without importing this package.
type Publisher struct {
	hub *Hub
}

func NewPublisher(hub *Hub) *Publisher { return &Publisher{hub: hub} }

func (p *Publisher) PublishMessageCreated(chatType string, chatID uuid.UUID, dto messages.DTO) {
	p.hub.publishToChat(chatType, chatID, encode(EventMessageCreated, dto))
}

func (p *Publisher) PublishMessageUpdated(chatType string, chatID uuid.UUID, dto messages.DTO) {
	p.hub.publishToChat(chatType, chatID, encode(EventMessageUpdated, dto))
}

func (p *Publisher) PublishMessageDeleted(chatType string, chatID, messageID uuid.UUID) {
	p.hub.publishToChat(chatType, chatID, encode(EventMessageDeleted, map[string]any{
		"chatType":  chatType,
		"chatId":    chatID,
		"messageId": messageID,
	}))
}

// PublishNotification pushes a notification.created event to one user's
// connected clients; satisfies the notifications.WSPublisher port.
func (p *Publisher) PublishNotification(userID uuid.UUID, data any) {
	p.hub.publishToUsers([]uuid.UUID{userID}, encode(EventNotification, data))
}

// PublishUserUpdated pushes a user's refreshed public profile to an audience
// (their chat partners and group co-members) so cached names/avatars update
// live; satisfies the users.ProfileBroadcaster port structurally.
func (p *Publisher) PublishUserUpdated(audience []uuid.UUID, profile any) {
	p.hub.publishToUsers(audience, encode(EventUserUpdated, profile))
}

// PublishBoardChanged tells a group's members their task board changed so
// they can refetch it; satisfies the boards.Publisher port. The group's
// members are the recipients (chatType "group" resolves to them).
func (p *Publisher) PublishBoardChanged(groupID uuid.UUID) {
	p.hub.publishToChat("group", groupID, encode(EventBoardChanged, map[string]any{"groupId": groupID}))
}

// PublishRatingChanged tells every connected client the shared rating board
// changed, so they refetch it; satisfies rating.ChangePublisher.
func (p *Publisher) PublishRatingChanged() {
	p.hub.broadcast(encode(EventRatingChanged, map[string]any{}))
}

// PublishPollChanged tells every connected client the voting board changed,
// so they refetch it; satisfies voting.ChangePublisher.
func (p *Publisher) PublishPollChanged() {
	p.hub.broadcast(encode(EventPollChanged, map[string]any{}))
}

// PublishGroupChanged tells a group's members the group's metadata (e.g. its
// avatar) changed so they can refetch it; satisfies groups.ChangePublisher.
func (p *Publisher) PublishGroupChanged(groupID uuid.UUID) {
	p.hub.publishToChat("group", groupID, encode(EventGroupChanged, map[string]any{"groupId": groupID}))
}

// --- call signaling relay (satisfies calls.CallPublisher structurally) ---
//
// Each method relays one server→client call event to a single user's connected
// clients on any node. Media is peer-to-peer; only signaling passes through here.

func (p *Publisher) Incoming(to, callID, fromID uuid.UUID, fromName string, fromAvatar *string, chatID uuid.UUID, sdp string) {
	p.hub.publishToUsers([]uuid.UUID{to}, encode(EventCallIncoming, callIncomingData{
		CallID: callID,
		From:   callFrom{ID: fromID, DisplayName: fromName, AvatarURL: fromAvatar},
		ChatID: chatID,
		SDP:    sdp,
	}))
}

func (p *Publisher) Answered(to, callID uuid.UUID, sdp string) {
	p.hub.publishToUsers([]uuid.UUID{to}, encode(EventCallAnswered, callAnsweredData{CallID: callID, SDP: sdp}))
}

func (p *Publisher) ICE(to, callID, from uuid.UUID, candidate json.RawMessage) {
	p.hub.publishToUsers([]uuid.UUID{to}, encode(EventCallICE, callICEData{CallID: callID, FromUserID: from, Candidate: candidate}))
}

func (p *Publisher) Rejected(to, callID uuid.UUID) {
	p.hub.publishToUsers([]uuid.UUID{to}, encode(EventCallRejected, callRefData{CallID: callID}))
}

func (p *Publisher) Canceled(to, callID uuid.UUID) {
	p.hub.publishToUsers([]uuid.UUID{to}, encode(EventCallCanceled, callRefData{CallID: callID}))
}

func (p *Publisher) Ended(to, callID uuid.UUID, reason string) {
	p.hub.publishToUsers([]uuid.UUID{to}, encode(EventCallEnded, callEndedData{CallID: callID, Reason: reason}))
}

func (p *Publisher) Busy(to, callID uuid.UUID) {
	p.hub.publishToUsers([]uuid.UUID{to}, encode(EventCallBusy, callRefData{CallID: callID}))
}

func (p *Publisher) Timeout(to, callID uuid.UUID) {
	p.hub.publishToUsers([]uuid.UUID{to}, encode(EventCallTimeout, callRefData{CallID: callID}))
}

// PublishReaction broadcasts a reaction change; satisfies the
// reactions.Publisher port structurally.
func (p *Publisher) PublishReaction(chatType string, chatID, messageID, userID uuid.UUID, emoji string, added bool) {
	event := EventReactionAdded
	if !added {
		event = EventReactionRemoved
	}
	p.hub.publishToChat(chatType, chatID, encode(event, reactionData{
		ChatType:  chatType,
		ChatID:    chatID,
		MessageID: messageID,
		UserID:    userID,
		Emoji:     emoji,
	}))
}
