package notifications

import (
	"context"
	"encoding/json"
	"regexp"

	"github.com/google/uuid"

	"kisy-backend/internal/messages"
)

// mentionPattern matches @username tokens using the same character class
// as the username validator.
var mentionPattern = regexp.MustCompile(`@([A-Za-z0-9_]{3,32})`)

// OnMessage implements messages.Notifier: it resolves @mentions in the
// message to chat recipients and raises a notification for each. It is
// best-effort — failures are swallowed so they never block message
// delivery.
func (s *Service) OnMessage(ctx context.Context, m messages.DTO) {
	if m.Text == nil || *m.Text == "" {
		return
	}

	matches := mentionPattern.FindAllStringSubmatch(*m.Text, -1)
	if len(matches) == 0 {
		return
	}

	// Recipients of this chat bound who may legitimately be mentioned; a
	// user cannot be pulled into a conversation they cannot see.
	recipients, err := s.recipients(ctx, m.ChatType, m.ChatID)
	if err != nil {
		return
	}
	allowed := make(map[uuid.UUID]struct{}, len(recipients))
	for _, r := range recipients {
		allowed[r] = struct{}{}
	}

	seen := make(map[uuid.UUID]struct{})
	for _, mt := range matches {
		username := mt[1]
		uid, ok := s.usernames(ctx, username)
		if !ok {
			continue
		}
		if uid == m.SenderID {
			continue // no self-notification
		}
		if _, isRecipient := allowed[uid]; !isRecipient {
			continue
		}
		if _, dup := seen[uid]; dup {
			continue
		}
		seen[uid] = struct{}{}

		s.raiseMention(ctx, uid, m)
	}
}

func (s *Service) raiseMention(ctx context.Context, target uuid.UUID, m messages.DTO) {
	payload, _ := json.Marshal(map[string]any{
		"chatType":  m.ChatType,
		"chatId":    m.ChatID,
		"messageId": m.ID,
		"senderId":  m.SenderID,
	})

	if err := s.repo.Create(ctx, s.pool, target, TypeMention, payload); err != nil {
		return
	}
	if s.mentions != nil {
		_ = s.mentions.AddMention(ctx, s.pool, m.ID, target)
	}
	if s.ws != nil {
		s.ws.PublishNotification(target, map[string]any{
			"type":      TypeMention,
			"chatType":  m.ChatType,
			"chatId":    m.ChatID,
			"messageId": m.ID,
			"senderId":  m.SenderID,
		})
	}
}
