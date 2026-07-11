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

// OnMessage implements messages.Notifier. It raises @mention notifications
// and (stage G) fans push out for new messages according to each recipient's
// mute state and group notification mode. Best-effort — failures are
// swallowed so they never block message delivery.
func (s *Service) OnMessage(ctx context.Context, m messages.DTO) {
	recipients, err := s.recipients(ctx, m.ChatType, m.ChatID)
	if err != nil {
		return
	}
	// Everyone but the sender is a candidate recipient.
	candidates := make([]uuid.UUID, 0, len(recipients))
	for _, r := range recipients {
		if r != m.SenderID {
			candidates = append(candidates, r)
		}
	}
	if len(candidates) == 0 {
		return
	}

	// Resolve which recipients are @mentioned (text messages only; E2EE
	// bodies are ciphertext and cannot be scanned server-side).
	allowed := make(map[uuid.UUID]struct{}, len(candidates))
	for _, r := range candidates {
		allowed[r] = struct{}{}
	}
	mentioned := s.resolveMentions(ctx, m, allowed)

	// Gate by mute + settings (stage G). Absent prefs = notify everyone.
	muted := map[uuid.UUID]struct{}{}
	settings := map[uuid.UUID]NotifSettings{}
	if s.prefs != nil {
		if mu, err := s.prefs.MutedUsers(ctx, m.ChatType, m.ChatID, candidates); err == nil {
			muted = mu
		}
		if st, err := s.prefs.SettingsFor(ctx, candidates); err == nil {
			settings = st
		}
	}

	for _, uid := range candidates {
		_, isMuted := muted[uid]
		_, isMentioned := mentioned[uid]

		// An @mention always raises the in-app notification (unless muted),
		// so the mention still appears in the bell even when push is off.
		if isMentioned && !isMuted {
			s.raiseMention(ctx, uid, m)
		}

		if isMuted {
			continue // muted chats: no push at all
		}

		set, hasSet := settings[uid]
		if !hasSet {
			set = NotifSettings{Sound: true, Preview: true, GroupMode: "all"}
		}
		if !s.shouldPush(m.ChatType, set.GroupMode, isMentioned) {
			continue
		}
		// Mentions get their own push inside raiseMention; here we push for
		// non-mention new messages (group "all" mode / private chats).
		if !isMentioned {
			s.pushNewMessage(ctx, uid, m, set)
		}
	}
}

// shouldPush decides whether a recipient should be pushed for this message.
// Private chats always notify; groups follow the recipient's group mode.
func (s *Service) shouldPush(chatType, groupMode string, isMentioned bool) bool {
	if chatType != messages.ChatGroup {
		return true // private chats always notify (when not muted)
	}
	switch groupMode {
	case "none":
		return false
	case "mentions_only":
		return isMentioned
	default: // "all"
		return true
	}
}

// resolveMentions maps @usernames in a text message to allowed recipient ids.
func (s *Service) resolveMentions(ctx context.Context, m messages.DTO, allowed map[uuid.UUID]struct{}) map[uuid.UUID]struct{} {
	out := map[uuid.UUID]struct{}{}
	if m.Text == nil || *m.Text == "" {
		return out
	}
	for _, mt := range mentionPattern.FindAllStringSubmatch(*m.Text, -1) {
		uid, ok := s.usernames(ctx, mt[1])
		if !ok || uid == m.SenderID {
			continue
		}
		if _, isRecipient := allowed[uid]; isRecipient {
			out[uid] = struct{}{}
		}
	}
	return out
}

func (s *Service) chatURL(m messages.DTO) string {
	if m.ChatType == messages.ChatGroup {
		return "/group/" + m.ChatID.String()
	}
	return "/chat/" + m.ChatID.String()
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
	if s.pusher != nil {
		go s.pusher.Notify(context.Background(), target, "KISY", "Вас упомянули в сообщении", s.chatURL(m))
	}
}

// pushNewMessage sends a content-less new-message push (stage G). The body is
// generic — message content stays private (E2EE and, by policy, plaintext
// too); the preview setting is reserved for future opt-in text previews.
func (s *Service) pushNewMessage(ctx context.Context, target uuid.UUID, m messages.DTO, _ NotifSettings) {
	if s.pusher == nil {
		return
	}
	go s.pusher.Notify(context.Background(), target, "KISY", "Новое сообщение", s.chatURL(m))
}
