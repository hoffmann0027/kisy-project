package messages

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"kisy-backend/internal/access"
	"kisy-backend/internal/audit"
	"kisy-backend/pkg/pagination"
)

const actionMessageDeleted = "message.deleted"

// ActorMeta identifies the acting user.
type ActorMeta struct {
	UserID    uuid.UUID
	RoleLevel int
	SessionID uuid.UUID
	IPHash    string
	RequestID string
}

// Authorizer decides whether an actor may read from and post to a chat.
// The two funcs are injected by the router: private chats delegate to the
// chats service (participation), groups to the groups service
// (membership). Both return the module's not-found error for hidden or
// missing chats so the message layer never leaks existence.
type Authorizer struct {
	Private func(ctx context.Context, chatID, actorID uuid.UUID) error
	Group   func(ctx context.Context, groupID uuid.UUID, actorID uuid.UUID, actorLevel int) error
}

// Publisher fans a persisted event out to connected clients. It is
// satisfied by the websocket hub and injected to avoid a messages→ws
// import cycle; a nil Publisher disables real-time delivery.
type Publisher interface {
	PublishMessageCreated(chatType string, chatID uuid.UUID, dto DTO)
	PublishMessageDeleted(chatType string, chatID, messageID uuid.UUID)
}

type Service struct {
	pool  *pgxpool.Pool
	repo  Repository
	audit audit.Recorder
	authz Authorizer
	pub   Publisher
}

func NewService(pool *pgxpool.Pool, repo Repository, rec audit.Recorder, authz Authorizer) *Service {
	return &Service{pool: pool, repo: repo, audit: rec, authz: authz}
}

// SetPublisher wires the real-time publisher after construction (the hub
// and this service are created together at startup).
func (s *Service) SetPublisher(p Publisher) { s.pub = p }

// authorize checks the actor may access the target chat, normalizing the
// chat type.
func (s *Service) authorize(ctx context.Context, chatType string, chatID uuid.UUID, actor ActorMeta) error {
	switch chatType {
	case ChatPrivate:
		return s.authz.Private(ctx, chatID, actor.UserID)
	case ChatGroup:
		return s.authz.Group(ctx, chatID, actor.UserID, actor.RoleLevel)
	default:
		return ErrBadChatType
	}
}

// SendInput is validated by the handler.
type SendInput struct {
	ChatType string
	ChatID   uuid.UUID
	Text     string
	ReplyTo  *uuid.UUID
}

// Send validates access, persists the message and publishes it.
func (s *Service) Send(ctx context.Context, in SendInput, actor ActorMeta) (*Message, error) {
	text := strings.TrimSpace(in.Text)
	if text == "" {
		return nil, ErrEmptyContent
	}
	if len(text) > MaxTextLength {
		text = text[:MaxTextLength]
	}

	if err := s.authorize(ctx, in.ChatType, in.ChatID, actor); err != nil {
		return nil, err
	}

	// A reply target must belong to the same chat, otherwise it could leak
	// message ids across conversations.
	if in.ReplyTo != nil {
		parent, err := s.repo.GetByID(ctx, s.pool, *in.ReplyTo)
		if err != nil {
			return nil, ErrNotFound
		}
		if parent.ChatType != in.ChatType || parent.ChatID != in.ChatID {
			return nil, ErrForbidden
		}
	}

	m := &Message{
		ChatType: in.ChatType,
		ChatID:   in.ChatID,
		SenderID: actor.UserID,
		Text:     &text,
		ReplyTo:  in.ReplyTo,
	}
	if err := s.repo.Create(ctx, s.pool, m); err != nil {
		return nil, err
	}

	if s.pub != nil {
		s.pub.PublishMessageCreated(m.ChatType, m.ChatID, m.ToDTO())
	}
	return m, nil
}

// List returns a page of messages for a chat the actor may access.
func (s *Service) List(ctx context.Context, chatType string, chatID uuid.UUID, cursor string, limit int, actor ActorMeta) (*pagination.Page[DTO], error) {
	if err := s.authorize(ctx, chatType, chatID, actor); err != nil {
		return nil, err
	}

	cur, present, err := pagination.Decode(cursor)
	if err != nil {
		return nil, err
	}
	var curPtr *pagination.Cursor
	if present {
		curPtr = &cur
	}

	limit = pagination.NormalizeLimit(limit)
	rows, err := s.repo.ListPage(ctx, s.pool, chatType, chatID, curPtr, limit)
	if err != nil {
		return nil, err
	}

	page := &pagination.Page[DTO]{Items: make([]DTO, 0, limit)}
	if len(rows) > limit {
		last := rows[limit-1]
		page.NextCursor = pagination.Encode(pagination.Cursor{CreatedAt: last.CreatedAt, ID: last.ID})
		page.HasMore = true
		rows = rows[:limit]
	}
	for i := range rows {
		page.Items = append(page.Items, rows[i].ToDTO())
	}
	return page, nil
}

// Delete removes a message per policy: the sender may delete their own
// message, and the CEO may delete any message. The deletion is audited and
// published.
func (s *Service) Delete(ctx context.Context, messageID uuid.UUID, actor ActorMeta) error {
	m, err := s.repo.GetByID(ctx, s.pool, messageID)
	if err != nil {
		return err
	}

	// The actor must be able to see the chat at all before we reveal
	// anything about the message.
	if err := s.authorize(ctx, m.ChatType, m.ChatID, actor); err != nil {
		return ErrNotFound
	}

	if m.SenderID != actor.UserID && !access.IsCEO(actor.RoleLevel) {
		return ErrForbidden
	}

	now := time.Now().UTC()
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("messages: begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	if err := s.repo.SoftDelete(ctx, tx, messageID, now); err != nil {
		return err
	}
	if err := s.audit.Record(ctx, tx, audit.Event{
		ActorID:    &actor.UserID,
		Action:     actionMessageDeleted,
		TargetType: "message",
		TargetID:   &messageID,
		IPHash:     actor.IPHash,
		SessionID:  &actor.SessionID,
		RequestID:  actor.RequestID,
		Metadata:   map[string]any{"chatType": m.ChatType, "chatId": m.ChatID.String()},
	}); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("messages: commit: %w", err)
	}

	if s.pub != nil {
		s.pub.PublishMessageDeleted(m.ChatType, m.ChatID, messageID)
	}
	return nil
}
