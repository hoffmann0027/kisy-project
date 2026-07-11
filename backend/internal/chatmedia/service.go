package chatmedia

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"kisy-backend/internal/attachments"
	"kisy-backend/pkg/pagination"
)

// Authorizer decides whether an actor may read a chat — the same injection
// pattern (and therefore the same clearance rules) as the message list.
type Authorizer struct {
	Private func(ctx context.Context, chatID, actorID uuid.UUID) error
	Group   func(ctx context.Context, groupID uuid.UUID, actorID uuid.UUID, actorLevel int) error
}

type Actor struct {
	UserID    uuid.UUID
	RoleLevel int
}

type Service struct {
	pool  *pgxpool.Pool
	repo  Repository
	authz Authorizer
}

func NewService(pool *pgxpool.Pool, repo Repository, authz Authorizer) *Service {
	return &Service{pool: pool, repo: repo, authz: authz}
}

func (s *Service) authorize(ctx context.Context, chatType string, chatID uuid.UUID, actor Actor) error {
	switch chatType {
	case "private":
		return s.authz.Private(ctx, chatID, actor.UserID)
	case "group":
		return s.authz.Group(ctx, chatID, actor.UserID, actor.RoleLevel)
	default:
		return ErrBadKind
	}
}

// MediaPage is one page of the media/files tabs.
type MediaPage struct {
	Items      []Item `json:"items"`
	NextCursor string `json:"nextCursor,omitempty"`
	HasMore    bool   `json:"hasMore"`
}

// LinkPage is one page of the links tab.
type LinkPage struct {
	Items      []LinkItem `json:"items"`
	NextCursor string     `json:"nextCursor,omitempty"`
	HasMore    bool       `json:"hasMore"`
}

func tabKinds(tab string) ([]string, bool) {
	switch tab {
	case TabMedia:
		return []string{attachments.KindImage, attachments.KindVideo}, true
	case TabFiles:
		return []string{attachments.KindFile}, true
	default:
		return nil, false
	}
}

// ListAttachments serves the media and files tabs.
func (s *Service) ListAttachments(ctx context.Context, chatType string, chatID uuid.UUID, tab, cursor string, limit int, actor Actor) (*MediaPage, error) {
	kinds, ok := tabKinds(tab)
	if !ok {
		return nil, ErrBadKind
	}
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

	rows, err := s.repo.ListAttachments(ctx, s.pool, chatType, chatID, kinds, curPtr, limit)
	if err != nil {
		return nil, err
	}
	page := &MediaPage{Items: []Item{}}
	if len(rows) > limit {
		last := rows[limit-1]
		page.NextCursor = pagination.Encode(pagination.Cursor{CreatedAt: last.CreatedAt, ID: last.Attachment.ID})
		page.HasMore = true
		rows = rows[:limit]
	}
	page.Items = append(page.Items, rows...)
	return page, nil
}

// ListLinks serves the links tab: plaintext messages are scanned for URLs.
// One message may contribute several links; pagination is by message.
func (s *Service) ListLinks(ctx context.Context, chatType string, chatID uuid.UUID, cursor string, limit int, actor Actor) (*LinkPage, error) {
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

	rows, err := s.repo.ListLinkMessages(ctx, s.pool, chatType, chatID, curPtr, limit)
	if err != nil {
		return nil, err
	}
	page := &LinkPage{Items: []LinkItem{}}
	if len(rows) > limit {
		last := rows[limit-1]
		page.NextCursor = pagination.Encode(last.CreatedAt)
		page.HasMore = true
		rows = rows[:limit]
	}
	for _, src := range rows {
		for _, u := range ExtractLinks(src.Text) {
			page.Items = append(page.Items, LinkItem{
				URL:       u,
				MessageID: src.MessageID,
				SenderID:  src.SenderID,
				CreatedAt: src.CreatedAt.CreatedAt,
			})
		}
	}
	return page, nil
}
