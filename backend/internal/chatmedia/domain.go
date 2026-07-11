// Package chatmedia aggregates a chat's shared content for the context
// panel tabs (stage C): media (images/video), files and links. Read-only
// over the messages/attachments tables; access is checked with the same
// chat authorizer the message list uses, so nothing leaks across
// clearances — an actor only ever sees content of chats they can open.
package chatmedia

import (
	"errors"
	"regexp"
	"time"

	"github.com/google/uuid"

	"kisy-backend/internal/attachments"
)

var (
	ErrNotFound  = errors.New("chatmedia: not found")
	ErrForbidden = errors.New("chatmedia: forbidden")
	ErrBadKind   = errors.New("chatmedia: unknown kind")
)

// Tab kinds.
const (
	TabMedia = "media" // image + video attachments
	TabFiles = "files" // plain file attachments (voice notes are not files)
	TabLinks = "links" // URLs extracted from plaintext message bodies
)

// Item is one entry of the media/files tabs: the attachment plus enough
// message context to jump to the original.
type Item struct {
	Attachment attachments.DTO `json:"attachment"`
	MessageID  uuid.UUID       `json:"messageId"`
	SenderID   uuid.UUID       `json:"senderId"`
	CreatedAt  time.Time       `json:"createdAt"`
}

// LinkItem is one entry of the links tab. In E2EE chats the server stores
// only ciphertext, so links are extractable server-side only from plaintext
// messages — an honest, documented limitation (client-side indexing of
// decrypted history is part of the E2EE roadmap).
type LinkItem struct {
	URL       string    `json:"url"`
	MessageID uuid.UUID `json:"messageId"`
	SenderID  uuid.UUID `json:"senderId"`
	CreatedAt time.Time `json:"createdAt"`
}

// urlPattern matches http(s) URLs inside message text. Trailing punctuation
// that is almost never part of a URL is trimmed by ExtractLinks.
var urlPattern = regexp.MustCompile(`https?://[^\s<>"']+`)

// ExtractLinks returns the unique URLs of one message body, in order.
func ExtractLinks(text string) []string {
	raw := urlPattern.FindAllString(text, -1)
	if len(raw) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(raw))
	out := make([]string, 0, len(raw))
	for _, u := range raw {
		u = trimTrailingPunct(u)
		if u == "" {
			continue
		}
		if _, dup := seen[u]; dup {
			continue
		}
		seen[u] = struct{}{}
		out = append(out, u)
	}
	return out
}

func trimTrailingPunct(u string) string {
	for len(u) > 0 {
		switch u[len(u)-1] {
		case '.', ',', ';', ':', '!', '?', ')', ']', '}':
			u = u[:len(u)-1]
		default:
			return u
		}
	}
	return u
}
