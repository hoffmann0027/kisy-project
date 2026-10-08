// Package announcements is notifications written by people: levels 1-3 tell
// everyone, every basic account, chosen levels or one person something.
//
// The rules, as the owner set them:
//
//   - Only downwards and sideways. An author at level L reaches levels L..10
//     and basic accounts; "everyone" means everyone at or below them. Only the
//     CEO reaches the whole company. The same rule as starting a private chat
//     (access.CanInitiateChat), so a director cannot page the CEO this way
//     either.
//   - Always delivered in full: the notifications list, the live event and a
//     push to the phone.
//   - Bounded, because a hijacked manager account is a spam cannon:
//     BroadcastsPerDay and PersonalPerDay for levels 2-3, none for the CEO.
//   - The author or the CEO can revoke one; it then leaves every recipient's
//     list (a push already shown on a phone cannot be taken back).
//
// The recipient always sees who wrote it, name and role: an announcement is a
// message from a person, never "from KISY", so it cannot be dressed up as one.
package announcements

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"kisy-backend/internal/access"
	"kisy-backend/internal/audit"
)

// Audiences.
const (
	AudienceAll    = "all"
	AudienceBasic  = "basic"
	AudienceLevels = "levels"
	AudienceUser   = "user"
)

// Limits.
const (
	MaxTitle = 100
	MaxBody  = 1000
	// AuthorMaxLevel: the lowest level that may write announcements.
	AuthorMaxLevel = 3
	// BroadcastsPerDay / PersonalPerDay bound levels 2-3 over any 24 hours.
	// Revoked announcements still count: revoking is not a way to resend.
	BroadcastsPerDay = 10
	PersonalPerDay   = 50
)

// NotificationType is the type of the notifications rows announcements write.
const NotificationType = "announcement"

// Audit actions.
const (
	ActionSent    = "announcement.sent"
	ActionRevoked = "announcement.revoked"
)

var (
	// ErrNotAuthor: the caller is not at levels 1-3.
	ErrNotAuthor = errors.New("announcements: only levels 1-3 may send announcements")
	// ErrValidation: a malformed request (audience, title, body, levels).
	ErrValidation = errors.New("announcements: invalid announcement")
	// ErrAboveYou: a requested level is above the author's own.
	ErrAboveYou = errors.New("announcements: cannot address a level above your own")
	// ErrUnreachable: the one person named cannot be notified by this author —
	// above them, blocked either way, or not an active account. One error for
	// all of these, so the answer says nothing about which.
	ErrUnreachable = errors.New("announcements: this person cannot be notified")
	// ErrQuota: the daily limit is spent.
	ErrQuota = errors.New("announcements: daily limit reached")
	// ErrNotFound: no such announcement, or not one the caller may revoke.
	ErrNotFound = errors.New("announcements: not found")
)

// ActorMeta identifies the caller.
type ActorMeta struct {
	UserID    uuid.UUID
	SessionID uuid.UUID
	IPHash    string
	RequestID string
}

// Input is a new announcement.
type Input struct {
	Audience string
	Levels   []int
	UserID   uuid.UUID
	Title    string
	Body     string
}

// Author is who wrote an announcement, as recipients see it.
type Author struct {
	ID          uuid.UUID `json:"id"`
	DisplayName string    `json:"displayName"`
	RoleLevel   int       `json:"roleLevel"`
}

// Announcement is one sent announcement, as its author (or the CEO) sees it.
type Announcement struct {
	ID             uuid.UUID  `json:"id"`
	Author         Author     `json:"author"`
	Audience       string     `json:"audience"`
	Levels         []int      `json:"levels,omitempty"`
	TargetUserID   *uuid.UUID `json:"targetUserId,omitempty"`
	TargetName     *string    `json:"targetName,omitempty"`
	Title          string     `json:"title"`
	Body           string     `json:"body"`
	RecipientCount int        `json:"recipientCount"`
	CreatedAt      time.Time  `json:"createdAt"`
	RevokedAt      *time.Time `json:"revokedAt,omitempty"`
}

// Publisher pushes live events to one user's open clients. Satisfied by
// *ws.Publisher.
type Publisher interface {
	PublishNotification(userID uuid.UUID, data any)
	PublishNotificationRevoked(userID uuid.UUID, data any)
}

// Pusher sends a push to a user's devices. Satisfied by *push.Service.
type Pusher interface {
	Notify(ctx context.Context, userID uuid.UUID, title, body, url string)
}

// pushWorkers bounds how many recipients are pushed to at once: an
// announcement to everyone must not open a connection per person in one go.
const pushWorkers = 8

type Service struct {
	pool   *pgxpool.Pool
	audit  audit.Recorder
	pub    Publisher
	pusher Pusher
	now    func() time.Time
}

func NewService(pool *pgxpool.Pool, rec audit.Recorder) *Service {
	return &Service{pool: pool, audit: rec, now: time.Now}
}

// SetPublisher wires live delivery.
func (s *Service) SetPublisher(p Publisher) { s.pub = p }

// SetPusher wires push delivery.
func (s *Service) SetPusher(p Pusher) { s.pusher = p }

func normalize(in Input) (Input, error) {
	in.Title = strings.TrimSpace(in.Title)
	in.Body = strings.TrimSpace(in.Body)
	if in.Title == "" || in.Body == "" ||
		utf8.RuneCountInString(in.Title) > MaxTitle || utf8.RuneCountInString(in.Body) > MaxBody {
		return in, ErrValidation
	}
	switch in.Audience {
	case AudienceAll, AudienceBasic:
		if len(in.Levels) > 0 || in.UserID != uuid.Nil {
			return in, ErrValidation
		}
	case AudienceLevels:
		if len(in.Levels) == 0 || in.UserID != uuid.Nil {
			return in, ErrValidation
		}
		seen := map[int]bool{}
		levels := make([]int, 0, len(in.Levels))
		for _, l := range in.Levels {
			if !access.HasLevel(l) {
				return in, ErrValidation
			}
			if !seen[l] {
				seen[l] = true
				levels = append(levels, l)
			}
		}
		in.Levels = levels
	case AudienceUser:
		if in.UserID == uuid.Nil || len(in.Levels) > 0 {
			return in, ErrValidation
		}
	default:
		return in, ErrValidation
	}
	return in, nil
}

// authorLevel reads the caller's live level (not the token's: a demotion
// takes effect at once) and locks their row, so two sends at the same moment
// cannot both pass the daily limit.
func authorLevel(ctx context.Context, tx pgx.Tx, id uuid.UUID) (level int, name string, err error) {
	var lvl *int
	err = tx.QueryRow(ctx, `
		SELECT r.level, u.display_name
		FROM users u LEFT JOIN roles r ON r.id = u.role_id
		WHERE u.id = $1 AND u.is_active
		FOR UPDATE OF u`, id).Scan(&lvl, &name)
	if errors.Is(err, pgx.ErrNoRows) {
		return access.NoLevel, "", nil
	}
	if err != nil {
		return 0, "", fmt.Errorf("announcements: load author: %w", err)
	}
	if lvl == nil {
		return access.NoLevel, name, nil
	}
	return *lvl, name, nil
}

// Send writes an announcement and delivers it. The rows are committed before
// anything goes out; the live event and the push are best effort.
func (s *Service) Send(ctx context.Context, actor ActorMeta, in Input) (*Announcement, error) {
	in, err := normalize(in)
	if err != nil {
		return nil, err
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	level, name, err := authorLevel(ctx, tx, actor.UserID)
	if err != nil {
		return nil, err
	}
	if !access.HasLevel(level) || level > AuthorMaxLevel {
		return nil, ErrNotAuthor
	}

	switch in.Audience {
	case AudienceLevels:
		for _, l := range in.Levels {
			if !access.CanInitiateChat(level, l) {
				return nil, ErrAboveYou
			}
		}
	case AudienceUser:
		if err := reachable(ctx, tx, actor.UserID, level, in.UserID); err != nil {
			return nil, err
		}
	}

	if !access.IsCEO(level) {
		personal := in.Audience == AudienceUser
		limit := BroadcastsPerDay
		if personal {
			limit = PersonalPerDay
		}
		var sent int
		if err := tx.QueryRow(ctx, `
			SELECT count(*) FROM announcements
			WHERE author_id = $1 AND created_at > $2 AND (audience = 'user') = $3`,
			actor.UserID, s.now().Add(-24*time.Hour), personal).Scan(&sent); err != nil {
			return nil, fmt.Errorf("announcements: quota: %w", err)
		}
		if sent >= limit {
			return nil, ErrQuota
		}
	}

	a := &Announcement{
		Author:   Author{ID: actor.UserID, DisplayName: name, RoleLevel: level},
		Audience: in.Audience,
		Title:    in.Title,
		Body:     in.Body,
	}
	// nil (SQL NULL) unless the audience is levels. pgx writes []int into the
	// smallint[] column itself, range-checked; normalize allowed only 1-10.
	var levels []int
	if in.Audience == AudienceLevels {
		a.Levels = in.Levels
		levels = in.Levels
	}
	var target *uuid.UUID
	if in.Audience == AudienceUser {
		t := in.UserID
		target = &t
		a.TargetUserID = target
	}
	if err := tx.QueryRow(ctx, `
		INSERT INTO announcements (author_id, audience, levels, target_user_id, title, body)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, created_at`,
		actor.UserID, in.Audience, levels, target, in.Title, in.Body).Scan(&a.ID, &a.CreatedAt); err != nil {
		return nil, fmt.Errorf("announcements: insert: %w", err)
	}

	payload := map[string]any{
		"announcementId": a.ID,
		"title":          a.Title,
		"body":           a.Body,
		"author":         a.Author,
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("announcements: encode payload: %w", err)
	}

	// One statement for every recipient. Active accounts only, never the
	// author, never above them, and never anyone with a block between the two.
	rows, err := tx.Query(ctx, `
		INSERT INTO notifications (user_id, type, payload, announcement_id)
		SELECT u.id, $1, $2, $3
		FROM users u LEFT JOIN roles r ON r.id = u.role_id
		WHERE u.is_active
		  AND u.id <> $4
		  AND (r.level IS NULL OR r.level >= $5)
		  AND NOT EXISTS (
		      SELECT 1 FROM user_blocks b
		      WHERE (b.blocker_id = u.id AND b.blocked_id = $4)
		         OR (b.blocker_id = $4 AND b.blocked_id = u.id))
		  AND CASE $6::text
		      WHEN 'all'    THEN true
		      WHEN 'basic'  THEN u.account_kind = 'basic'
		      WHEN 'levels' THEN r.level = ANY($7::smallint[])
		      WHEN 'user'   THEN u.id = $8
		      ELSE false
		  END
		RETURNING user_id`,
		NotificationType, raw, a.ID, actor.UserID, level, in.Audience, levels, target)
	if err != nil {
		return nil, fmt.Errorf("announcements: fan out: %w", err)
	}
	recipients, err := pgx.CollectRows(rows, pgx.RowTo[uuid.UUID])
	if err != nil {
		return nil, fmt.Errorf("announcements: fan out: %w", err)
	}
	a.RecipientCount = len(recipients)

	if _, err := tx.Exec(ctx, `UPDATE announcements SET recipient_count = $2 WHERE id = $1`, a.ID, a.RecipientCount); err != nil {
		return nil, fmt.Errorf("announcements: count: %w", err)
	}
	meta := map[string]any{"audience": in.Audience, "recipients": a.RecipientCount}
	if in.Audience == AudienceLevels {
		meta["levels"] = in.Levels
	}
	if err := s.audit.Record(ctx, tx, audit.Event{
		ActorID: &actor.UserID, Action: ActionSent, TargetType: "announcement", TargetID: &a.ID,
		IPHash: actor.IPHash, SessionID: &actor.SessionID, RequestID: actor.RequestID, Metadata: meta,
	}); err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	s.deliver(recipients, a, payload)
	return a, nil
}

// reachable: the one person named exists, is active, is not above the author
// and has no block with them either way.
func reachable(ctx context.Context, tx pgx.Tx, author uuid.UUID, authorLevel int, target uuid.UUID) error {
	if target == author {
		return ErrUnreachable
	}
	var ok bool
	err := tx.QueryRow(ctx, `
		SELECT (r.level IS NULL OR r.level >= $3)
		   AND NOT EXISTS (
		       SELECT 1 FROM user_blocks b
		       WHERE (b.blocker_id = u.id AND b.blocked_id = $2)
		          OR (b.blocker_id = $2 AND b.blocked_id = u.id))
		FROM users u LEFT JOIN roles r ON r.id = u.role_id
		WHERE u.id = $1 AND u.is_active`, target, author, authorLevel).Scan(&ok)
	if errors.Is(err, pgx.ErrNoRows) || (err == nil && !ok) {
		return ErrUnreachable
	}
	if err != nil {
		return fmt.Errorf("announcements: load recipient: %w", err)
	}
	return nil
}

// deliver sends the live event to everyone at once and the push through a
// small pool of workers, after the request has been answered.
func (s *Service) deliver(recipients []uuid.UUID, a *Announcement, payload map[string]any) {
	s.fanOut(recipients, NotificationType, payload, a.Title, a.Author.DisplayName+": "+a.Body, "/")
}

// fanOut sends a stored notification live to everyone at once and as a push
// through a small pool of workers, after the request has been answered.
func (s *Service) fanOut(recipients []uuid.UUID, typ string, payload map[string]any, title, body, url string) {
	if s.pub != nil {
		live := map[string]any{"type": typ}
		for k, v := range payload {
			live[k] = v
		}
		for _, id := range recipients {
			s.pub.PublishNotification(id, live)
		}
	}
	if s.pusher == nil || len(recipients) == 0 {
		return
	}
	// #nosec G118 -- deliberate: the push must outlive the request that sent
	// it. Each Notify is bounded by push.notifyTimeout.
	go func() {
		jobs := make(chan uuid.UUID)
		var wg sync.WaitGroup
		for i := 0; i < pushWorkers; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for id := range jobs {
					s.pusher.Notify(context.Background(), id, title, body, url)
				}
			}()
		}
		for _, id := range recipients {
			jobs <- id
		}
		close(jobs)
		wg.Wait()
	}()
}

// Revoke takes an announcement back: it leaves every recipient's list. Only
// its author or the CEO may; to anyone else it does not exist.
func (s *Service) Revoke(ctx context.Context, actor ActorMeta, id uuid.UUID) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	level, _, err := authorLevel(ctx, tx, actor.UserID)
	if err != nil {
		return err
	}
	var author uuid.UUID
	var revokedAt *time.Time
	err = tx.QueryRow(ctx, `SELECT author_id, revoked_at FROM announcements WHERE id = $1 FOR UPDATE`, id).Scan(&author, &revokedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		return fmt.Errorf("announcements: load: %w", err)
	}
	if author != actor.UserID && !access.IsCEO(level) {
		return ErrNotFound
	}
	if revokedAt != nil {
		return nil // already taken back
	}

	rows, err := tx.Query(ctx, `DELETE FROM notifications WHERE announcement_id = $1 RETURNING user_id`, id)
	if err != nil {
		return fmt.Errorf("announcements: revoke: %w", err)
	}
	recipients, err := pgx.CollectRows(rows, pgx.RowTo[uuid.UUID])
	if err != nil {
		return fmt.Errorf("announcements: revoke: %w", err)
	}
	if _, err := tx.Exec(ctx, `UPDATE announcements SET revoked_at = now(), revoked_by = $2 WHERE id = $1`, id, actor.UserID); err != nil {
		return fmt.Errorf("announcements: revoke: %w", err)
	}
	if err := s.audit.Record(ctx, tx, audit.Event{
		ActorID: &actor.UserID, Action: ActionRevoked, TargetType: "announcement", TargetID: &id,
		IPHash: actor.IPHash, SessionID: &actor.SessionID, RequestID: actor.RequestID,
		Metadata: map[string]any{"author": author, "removed": len(recipients)},
	}); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return err
	}
	if s.pub != nil {
		for _, uid := range recipients {
			s.pub.PublishNotificationRevoked(uid, map[string]any{"announcementId": id})
		}
	}
	return nil
}

// List returns sent announcements, newest first: the caller's own, or every
// one for the CEO, who can revoke any of them.
func (s *Service) List(ctx context.Context, actor ActorMeta, limit int) ([]Announcement, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	var level *int
	if err := s.pool.QueryRow(ctx, `
		SELECT r.level FROM users u LEFT JOIN roles r ON r.id = u.role_id WHERE u.id = $1`,
		actor.UserID).Scan(&level); err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return nil, fmt.Errorf("announcements: load caller: %w", err)
	}
	all := level != nil && access.IsCEO(*level)

	rows, err := s.pool.Query(ctx, `
		SELECT a.id, a.author_id, au.display_name, COALESCE(ar.level, 0),
		       a.audience, a.levels, a.target_user_id, t.display_name,
		       a.title, a.body, a.recipient_count, a.created_at, a.revoked_at
		FROM announcements a
		JOIN users au ON au.id = a.author_id
		LEFT JOIN roles ar ON ar.id = au.role_id
		LEFT JOIN users t ON t.id = a.target_user_id
		WHERE $1 OR a.author_id = $2
		ORDER BY a.created_at DESC
		LIMIT $3`, all, actor.UserID, limit)
	if err != nil {
		return nil, fmt.Errorf("announcements: list: %w", err)
	}
	defer rows.Close()
	out := []Announcement{}
	for rows.Next() {
		var a Announcement
		if err := rows.Scan(&a.ID, &a.Author.ID, &a.Author.DisplayName, &a.Author.RoleLevel,
			&a.Audience, &a.Levels, &a.TargetUserID, &a.TargetName,
			&a.Title, &a.Body, &a.RecipientCount, &a.CreatedAt, &a.RevokedAt); err != nil {
			return nil, fmt.Errorf("announcements: scan: %w", err)
		}
		out = append(out, a)
	}
	return out, rows.Err()
}
