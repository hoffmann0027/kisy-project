package ws

import (
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

// PublishBoardChanged tells a group's members their task board changed so
// they can refetch it; satisfies the boards.Publisher port. The group's
// members are the recipients (chatType "group" resolves to them).
func (p *Publisher) PublishBoardChanged(groupID uuid.UUID) {
	p.hub.publishToChat("group", groupID, encode(EventBoardChanged, map[string]any{"groupId": groupID}))
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
