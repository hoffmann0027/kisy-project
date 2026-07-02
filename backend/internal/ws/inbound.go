package ws

import (
	"context"
	"encoding/json"

	"kisy-backend/internal/messages"
)

// handleInbound dispatches a client frame. Unknown or malformed frames are
// answered with an error event rather than closing the connection.
func (h *Hub) handleInbound(c *Client, raw []byte) {
	var in Inbound
	if err := json.Unmarshal(raw, &in); err != nil {
		c.enqueue(encode(EventError, map[string]string{"message": "malformed frame"}))
		return
	}

	ctx := context.Background()
	switch in.Type {
	case TypeMessageSend:
		var p sendPayload
		if err := json.Unmarshal(in.Data, &p); err != nil {
			c.enqueue(encode(EventError, map[string]string{"message": "invalid message.send payload"}))
			return
		}
		h.handleSend(ctx, c, p)

	case TypeTypingStart:
		h.handleTyping(ctx, c, in.Data, EventTypingStarted)
	case TypeTypingStop:
		h.handleTyping(ctx, c, in.Data, EventTypingStopped)

	case TypeReadConfirmation:
		h.handleRead(ctx, c, in.Data)

	case TypePresenceSubscribe:
		var p subscribePayload
		if err := json.Unmarshal(in.Data, &p); err != nil {
			c.enqueue(encode(EventError, map[string]string{"message": "invalid presence.subscribe payload"}))
			return
		}
		h.subscribePresence(c, p.UserIDs)

	default:
		c.enqueue(encode(EventError, map[string]string{"message": "unknown message type"}))
	}
}

// handleSend persists a socket-sent message via the messages service. On
// success the service publishes message.created through this hub, so no
// extra broadcast is needed here; only the sender is told of failures.
func (h *Hub) handleSend(ctx context.Context, c *Client, p sendPayload) {
	if h.sender == nil {
		return
	}
	_, err := h.sender.Send(ctx, messages.SendInput{
		ChatType: p.ChatType,
		ChatID:   p.ChatID,
		Text:     p.Text,
		ReplyTo:  p.ReplyTo,
	}, messages.ActorMeta{
		UserID:    c.userID,
		RoleLevel: c.roleLevel,
		SessionID: c.sessionID,
	})
	if err != nil {
		c.enqueue(encode(EventError, map[string]string{"message": "message rejected"}))
	}
}

func (h *Hub) handleTyping(ctx context.Context, c *Client, data json.RawMessage, event string) {
	var ref chatRef
	if err := json.Unmarshal(data, &ref); err != nil {
		return
	}
	if h.authorizeChat != nil {
		if err := h.authorizeChat(ctx, ref.ChatType, ref.ChatID, c.userID, c.roleLevel); err != nil {
			return // silently drop typing for chats the actor cannot access
		}
	}
	frame := encode(event, typingData{ChatType: ref.ChatType, ChatID: ref.ChatID, UserID: c.userID})
	h.publishToChat(ref.ChatType, ref.ChatID, frame)
}

func (h *Hub) handleRead(ctx context.Context, c *Client, data json.RawMessage) {
	var p readPayload
	if err := json.Unmarshal(data, &p); err != nil {
		return
	}
	if h.authorizeChat != nil {
		if err := h.authorizeChat(ctx, p.ChatType, p.ChatID, c.userID, c.roleLevel); err != nil {
			return
		}
	}
	// Persist the read position (best-effort) so unread counters reflect
	// reads that arrive over the socket, not just via REST.
	if h.onRead != nil {
		h.onRead(ctx, c.userID, p.ChatType, p.ChatID, p.MessageID)
	}
	frame := encode(EventMessageRead, readData{
		ChatType:  p.ChatType,
		ChatID:    p.ChatID,
		MessageID: p.MessageID,
		UserID:    c.userID,
	})
	h.publishToChat(p.ChatType, p.ChatID, frame)
}
