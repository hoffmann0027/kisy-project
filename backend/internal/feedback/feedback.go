// Package feedback owns "Отзывы и предложения": a private line from anyone to
// the company's leadership.
//
// The rules, as the owner set them (October 2026):
//
//   - An author sees only their own entries, each with its reply once there
//     is one. Feedback used to be a public board; what someone tells the
//     leadership is not everyone's to read.
//   - One entry per author in any 24 hours.
//   - Levels 1-3 see every entry nobody has answered yet (the inbox) and reply
//     once. An answered entry leaves the inbox; its author is notified.
//   - Only the CEO may delete an entry.
//
// Who counts as levels 1-3 is read live from the database, not from the
// token, so a demotion takes effect at once. All of it is enforced here,
// never merely hidden in the UI.
package feedback

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"kisy-backend/internal/access"
	"kisy-backend/internal/i18n"
	"kisy-backend/internal/notifications"
	"kisy-backend/internal/platform/db"
)

const (
	maxBodyLen     = 2000
	defaultPageLen = 20
	maxPageLen     = 50

	// StaffMaxLevel: the lowest level that sees the inbox and replies.
	StaffMaxLevel = 3
	// Window: one author writes at most once in any such stretch.
	Window = 24 * time.Hour

	// NotificationType is what the author receives when answered.
	NotificationType = "feedback_reply"
)

// Scopes of List.
const (
	ScopeMine  = "mine"
	ScopeInbox = "inbox"
)

var (
	ErrNotFound       = errors.New("feedback: not found")
	ErrForbidden      = errors.New("feedback: not permitted")
	ErrEmpty          = errors.New("feedback: body is empty")
	ErrTooLong        = errors.New("feedback: body too long")
	ErrAlreadyReplied = errors.New("feedback: already answered")
)

// DailyLimitError: the author already wrote within the last 24 hours.
type DailyLimitError struct {
	RetryAfter time.Duration
}

func (e *DailyLimitError) Error() string { return "feedback: one entry per day" }

// Author is the public identity of a person in the feedback card.
type Author struct {
	ID          uuid.UUID `json:"id"`
	DisplayName string    `json:"displayName"`
	Username    string    `json:"username,omitempty"`
	AvatarURL   *string   `json:"avatarUrl"`
	// Null for an account outside the hierarchy.
	RoleLevel *int `json:"roleLevel"`
}

// Reply is leadership's answer to an entry.
type Reply struct {
	Body string    `json:"body"`
	At   time.Time `json:"at"`
	// By is nil when the person who replied has since been removed.
	By *Author `json:"by"`
}

// DTO is the API representation of one feedback entry.
type DTO struct {
	ID        uuid.UUID `json:"id"`
	Body      string    `json:"body"`
	Author    Author    `json:"author"`
	CreatedAt time.Time `json:"createdAt"`
	Reply     *Reply    `json:"reply"`
}

// Page is a cursor-paginated slice of feedback, newest first.
type Page struct {
	Items      []DTO   `json:"items"`
	NextCursor *string `json:"nextCursor"`
	HasMore    bool    `json:"hasMore"`
}

// Filter narrows List: one author's entries, or only unanswered ones.
type Filter struct {
	AuthorID       *uuid.UUID
	UnansweredOnly bool
}

// Repository is the persistence port for feedback.
type Repository interface {
	Create(ctx context.Context, q db.DBTX, authorID uuid.UUID, body string) (DTO, error)
	List(ctx context.Context, q db.DBTX, f Filter, before *time.Time, limit int) ([]DTO, error)
	Delete(ctx context.Context, q db.DBTX, id uuid.UUID) error
}

type PostgresRepository struct{}

func NewPostgresRepository() *PostgresRepository { return &PostgresRepository{} }

const selectColumns = `
	f.id, f.body, f.created_at,
	u.id, u.display_name, u.username, u.avatar_url, u.role_id,
	f.reply, f.replied_at,
	ru.id, ru.display_name, ru.username, ru.avatar_url, ru.role_id`

const fromClause = `
	FROM feedback f
	JOIN users u ON u.id = f.author_id
	LEFT JOIN users ru ON ru.id = f.replied_by`

func scan(row pgx.Row) (DTO, error) {
	var d DTO
	var reply *string
	var repliedAt *time.Time
	var rID *uuid.UUID
	var rName, rUsername *string
	var rAvatar *string
	var rLevel *int
	if err := row.Scan(&d.ID, &d.Body, &d.CreatedAt,
		&d.Author.ID, &d.Author.DisplayName, &d.Author.Username, &d.Author.AvatarURL, &d.Author.RoleLevel,
		&reply, &repliedAt,
		&rID, &rName, &rUsername, &rAvatar, &rLevel); err != nil {
		return DTO{}, err
	}
	if reply != nil && repliedAt != nil {
		d.Reply = &Reply{Body: *reply, At: *repliedAt}
		if rID != nil {
			d.Reply.By = &Author{ID: *rID, DisplayName: deref(rName), AvatarURL: rAvatar, RoleLevel: rLevel}
		}
	}
	return d, nil
}

func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func (r *PostgresRepository) Create(ctx context.Context, q db.DBTX, authorID uuid.UUID, body string) (DTO, error) {
	row := q.QueryRow(ctx, `
		WITH f AS (
			INSERT INTO feedback (author_id, body) VALUES ($1, $2)
			RETURNING id, body, created_at, author_id, reply, replied_at, replied_by
		)
		SELECT `+selectColumns+`
		FROM f
		JOIN users u ON u.id = f.author_id
		LEFT JOIN users ru ON ru.id = f.replied_by`,
		authorID, body)
	d, err := scan(row)
	if err != nil {
		return DTO{}, fmt.Errorf("feedback: create: %w", err)
	}
	return d, nil
}

func (r *PostgresRepository) List(ctx context.Context, q db.DBTX, f Filter, before *time.Time, limit int) ([]DTO, error) {
	rows, err := q.Query(ctx, `
		SELECT `+selectColumns+fromClause+`
		WHERE ($1::timestamptz IS NULL OR f.created_at < $1)
		  AND ($2::uuid IS NULL OR f.author_id = $2)
		  AND (NOT $3 OR f.replied_at IS NULL)
		ORDER BY f.created_at DESC, f.id DESC
		LIMIT $4`,
		before, f.AuthorID, f.UnansweredOnly, limit)
	if err != nil {
		return nil, fmt.Errorf("feedback: list: %w", err)
	}
	defer rows.Close()

	var out []DTO
	for rows.Next() {
		d, err := scan(rows)
		if err != nil {
			return nil, fmt.Errorf("feedback: scan: %w", err)
		}
		out = append(out, d)
	}
	return out, rows.Err()
}

func (r *PostgresRepository) Delete(ctx context.Context, q db.DBTX, id uuid.UUID) error {
	tag, err := q.Exec(ctx, `DELETE FROM feedback WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("feedback: delete: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// Actor identifies the acting user for authorization.
type Actor struct {
	UserID    uuid.UUID
	RoleLevel int
}

// Notifier tells an author their entry was answered. Satisfied by
// *notifications.Service.
type Notifier interface {
	Announce(ctx context.Context, recipients []uuid.UUID, a notifications.Announcement) error
}

type Service struct {
	pool     *pgxpool.Pool
	repo     Repository
	notifier Notifier
	now      func() time.Time
}

func NewService(pool *pgxpool.Pool, repo Repository) *Service {
	return &Service{pool: pool, repo: repo, now: time.Now}
}

// SetNotifier wires the "your feedback was answered" notification.
func (s *Service) SetNotifier(n Notifier) { s.notifier = n }

// liveLevel is the caller's level as the database has it now.
func liveLevel(ctx context.Context, q db.DBTX, userID uuid.UUID) (int, error) {
	var lvl *int
	err := q.QueryRow(ctx, `
		SELECT r.level FROM users u LEFT JOIN roles r ON r.id = u.role_id
		WHERE u.id = $1 AND u.is_active`, userID).Scan(&lvl)
	if errors.Is(err, pgx.ErrNoRows) || (err == nil && lvl == nil) {
		return access.NoLevel, nil
	}
	if err != nil {
		return 0, fmt.Errorf("feedback: load level: %w", err)
	}
	return *lvl, nil
}

// IsStaff reports whether a level sees the inbox and may reply.
func IsStaff(level int) bool { return access.HasLevel(level) && level <= StaffMaxLevel }

// Create validates and stores a feedback entry, at most one per author in
// any 24 hours.
func (s *Service) Create(ctx context.Context, authorID uuid.UUID, body string) (DTO, error) {
	body = strings.TrimSpace(body)
	if body == "" {
		return DTO{}, ErrEmpty
	}
	if len([]rune(body)) > maxBodyLen {
		return DTO{}, ErrTooLong
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return DTO{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// Two submissions at the same moment must not both pass the check.
	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended('feedback:' || $1::text, 0))`, authorID); err != nil {
		return DTO{}, fmt.Errorf("feedback: lock: %w", err)
	}
	var last *time.Time
	if err := tx.QueryRow(ctx, `SELECT max(created_at) FROM feedback WHERE author_id = $1`, authorID).Scan(&last); err != nil {
		return DTO{}, fmt.Errorf("feedback: last entry: %w", err)
	}
	if last != nil {
		if next := last.Add(Window); s.now().Before(next) {
			return DTO{}, &DailyLimitError{RetryAfter: next.Sub(s.now())}
		}
	}

	d, err := s.repo.Create(ctx, tx, authorID, body)
	if err != nil {
		return DTO{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return DTO{}, err
	}
	return d, nil
}

// List returns a page of feedback, newest first. ScopeInbox is the
// unanswered entries of everyone, for levels 1-3 only; anything else — and
// any request from someone outside levels 1-3 — is the caller's own entries.
// cursor is the createdAt of the last item already seen (RFC3339).
func (s *Service) List(ctx context.Context, viewer Actor, scope, cursor string, limit int) (Page, error) {
	f := Filter{AuthorID: &viewer.UserID}
	if scope == ScopeInbox {
		level, err := liveLevel(ctx, s.pool, viewer.UserID)
		if err != nil {
			return Page{}, err
		}
		if !IsStaff(level) {
			return Page{}, ErrForbidden
		}
		f = Filter{UnansweredOnly: true}
	}

	if limit <= 0 || limit > maxPageLen {
		limit = defaultPageLen
	}
	var before *time.Time
	if cursor != "" {
		if t, err := time.Parse(time.RFC3339Nano, cursor); err == nil {
			before = &t
		}
	}
	items, err := s.repo.List(ctx, s.pool, f, before, limit+1)
	if err != nil {
		return Page{}, err
	}
	page := Page{Items: items}
	if len(items) > limit {
		page.Items = items[:limit]
		last := page.Items[len(page.Items)-1].CreatedAt.Format(time.RFC3339Nano)
		page.NextCursor = &last
		page.HasMore = true
	}
	if page.Items == nil {
		page.Items = []DTO{}
	}
	return page, nil
}

// Reply answers an entry, once. Levels 1-3 only. The entry leaves the inbox
// and its author is notified.
func (s *Service) Reply(ctx context.Context, actor Actor, id uuid.UUID, body string) error {
	body = strings.TrimSpace(body)
	if body == "" {
		return ErrEmpty
	}
	if len([]rune(body)) > maxBodyLen {
		return ErrTooLong
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	level, err := liveLevel(ctx, tx, actor.UserID)
	if err != nil {
		return err
	}
	if !IsStaff(level) {
		return ErrForbidden
	}
	var author uuid.UUID
	var entry string
	var repliedAt *time.Time
	err = tx.QueryRow(ctx, `SELECT author_id, body, replied_at FROM feedback WHERE id = $1 FOR UPDATE`, id).
		Scan(&author, &entry, &repliedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		return fmt.Errorf("feedback: load: %w", err)
	}
	if repliedAt != nil {
		return ErrAlreadyReplied
	}
	if _, err := tx.Exec(ctx, `
		UPDATE feedback SET reply = $2, replied_at = now(), replied_by = $3 WHERE id = $1`,
		id, body, actor.UserID); err != nil {
		return fmt.Errorf("feedback: reply: %w", err)
	}
	var name string
	if err := tx.QueryRow(ctx, `SELECT display_name FROM users WHERE id = $1`, actor.UserID).Scan(&name); err != nil {
		return fmt.Errorf("feedback: load replier: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return err
	}

	if s.notifier != nil && author != actor.UserID {
		// The answer is in force; a lost notification only means the author
		// finds it next time they open the window.
		_ = s.notifier.Announce(ctx, []uuid.UUID{author}, notifications.Announcement{
			Type: NotificationType,
			Payload: map[string]any{
				"feedbackId": id,
				"feedback":   excerpt(entry, 120),
				"reply":      body,
				"by":         map[string]any{"id": actor.UserID, "displayName": name, "roleLevel": level},
			},
			PushTitle: i18n.M("feedback.replyPushTitle"),
			PushBody:  i18n.Raw(name + ": " + excerpt(body, 160)),
			URL:       "/hub",
		})
	}
	return nil
}

func excerpt(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return strings.TrimSpace(string(r[:n])) + "…"
}

// Delete removes a feedback entry. Only the CEO may do so.
func (s *Service) Delete(ctx context.Context, id uuid.UUID, actor Actor) error {
	level, err := liveLevel(ctx, s.pool, actor.UserID)
	if err != nil {
		return err
	}
	if !access.IsCEO(level) {
		return ErrForbidden
	}
	return s.repo.Delete(ctx, s.pool, id)
}
