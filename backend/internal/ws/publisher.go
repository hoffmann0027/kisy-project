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
