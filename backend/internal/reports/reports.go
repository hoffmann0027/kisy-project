// Package reports is how a person tells us about content or behaviour, and
// how the CEO works through what they told us (audit E-02).
//
// Two things shape it:
//
//   - The server cannot read private messages, and this package does not
//     pretend otherwise. A report about one carries its id and nothing else;
//     the queue says so out loud instead of showing an empty text field.
//   - Five different people reporting the same post hide it from the feed
//     until the CEO decides. One person cannot hide anything, and the author
//     still sees their own post — otherwise a small group could silence anyone
//     by pressing a button each.
package reports

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"kisy-backend/internal/audit"
)

// AutoHideThreshold: how many different people must report one post before it
// leaves the feed pending a decision. Also written into the SQL that filters
// the feed (internal/posts/repository.go) — change both together.
const AutoHideThreshold = 5

var (
	// ErrBadTarget: unknown kind of thing to report.
	ErrBadTarget = errors.New("reports: unknown target")
	// ErrBadReason: reason outside the list.
	ErrBadReason = errors.New("reports: unknown reason")
	// ErrSelf: reporting yourself is not a thing.
	ErrSelf = errors.New("reports: cannot report yourself")
	// ErrNotFound: no such report.
	ErrNotFound = errors.New("reports: not found")
)

// Kinds of thing a report can be about.
const (
	TargetUser      = "user"
	TargetMessage   = "message"
	TargetPost      = "post"
	TargetCommunity = "community"
)

// Reasons offered to the person reporting.
var reasons = map[string]bool{"spam": true, "abuse": true, "fraud": true, "illegal": true, "other": true}

// Statuses of a report in the queue.
const (
	StatusOpen     = "open"
	StatusResolved = "resolved"
	StatusRejected = "rejected"
)

// Report is one entry of the queue.
type Report struct {
	ID          uuid.UUID  `json:"id"`
	ReporterID  uuid.UUID  `json:"reporterId"`
	TargetKind  string     `json:"targetKind"`
	TargetID    uuid.UUID  `json:"targetId"`
	TargetOwner *uuid.UUID `json:"targetOwner"`
	Reason      string     `json:"reason"`
	Comment     string     `json:"comment,omitempty"`
	Status      string     `json:"status"`
	CreatedAt   time.Time  `json:"createdAt"`
	// SameTarget: how many open reports this thing has in total.
	SameTarget int `json:"sameTarget"`
	// AgainstOwner: how many open reports its author has in total.
	AgainstOwner int `json:"againstOwner"`
	// Content is what the thing says, when the server can read it: a post or a
	// group message. Nil for a private message — it is encrypted.
	Content *string `json:"content"`
	// Readable is false when the content exists but the server cannot read it.
	Readable bool `json:"readable"`
}

// OwnerResolver says who authored the thing being reported, so the queue can
// count reports against a person. Injected to keep this package out of posts
// and messages.
type OwnerResolver func(ctx context.Context, kind string, id uuid.UUID) (owner uuid.UUID, found bool, err error)

// VisibilityChecker says whether the reporter can see the thing being
// reported at all. Injected for the same reason as OwnerResolver.
type VisibilityChecker func(ctx context.Context, reporterID uuid.UUID, kind string, id uuid.UUID) (bool, error)

// WeightFunc says whether this reporter's reports move the automatic hiding of
// a post (AutoHideThreshold). False for an account still in its new-account
// quarantine: its report still reaches the queue, it just does not count.
type WeightFunc func(ctx context.Context, reporterID uuid.UUID) (bool, error)

type Service struct {
	pool    *pgxpool.Pool
	audit   audit.Recorder
	owner   OwnerResolver
	content ContentLoader
	visible VisibilityChecker
	weight  WeightFunc
}

func NewService(pool *pgxpool.Pool, rec audit.Recorder) *Service {
	return &Service{pool: pool, audit: rec}
}

// SetOwnerResolver wires the lookup of who authored a reported thing.
func (s *Service) SetOwnerResolver(f OwnerResolver) { s.owner = f }

// SetVisibility wires the check that a reporter can see what they report.
// Without it every report is refused: a report on something you cannot see
// is a way to probe for it, and — for a post — a vote toward hiding it.
func (s *Service) SetVisibility(f VisibilityChecker) { s.visible = f }

// SetWeight wires which reporters count toward hiding a post automatically.
// Unset, every report counts.
func (s *Service) SetWeight(f WeightFunc) { s.weight = f }

// ActorMeta is who is reporting.
type ActorMeta struct {
	UserID    uuid.UUID
	SessionID uuid.UUID
	IPHash    string
	RequestID string
}

// Input is a report as the app sends it.
type Input struct {
	TargetKind string
	TargetID   uuid.UUID
	Reason     string
	Comment    string
}

// MaxCommentLength bounds the free-text part.
const MaxCommentLength = 2000

// Create files a report. Reporting the same thing twice is not an error and
// does not count twice — the person pressed the button, and the outcome they
// asked for holds.
func (s *Service) Create(ctx context.Context, actor ActorMeta, in Input) (uuid.UUID, error) {
	switch in.TargetKind {
	case TargetUser, TargetMessage, TargetPost, TargetCommunity:
	default:
		return uuid.Nil, ErrBadTarget
	}
	if !reasons[in.Reason] {
		return uuid.Nil, ErrBadReason
	}
	if in.TargetKind == TargetUser && in.TargetID == actor.UserID {
		return uuid.Nil, ErrSelf
	}
	comment := strings.TrimSpace(in.Comment)
	if len([]rune(comment)) > MaxCommentLength {
		comment = string([]rune(comment)[:MaxCommentLength])
	}

	// Only what you can see can be reported. Anything else answers exactly as
	// a target that does not exist, so the endpoint tells nothing about posts
	// in closed communities or messages in other people's chats — and five
	// strangers cannot hide a post they were never shown.
	if s.visible == nil {
		return uuid.Nil, ErrNotFound
	}
	visible, err := s.visible(ctx, actor.UserID, in.TargetKind, in.TargetID)
	if err != nil {
		return uuid.Nil, err
	}
	if !visible {
		return uuid.Nil, ErrNotFound
	}

	counts := true
	if s.weight != nil {
		if counts, err = s.weight(ctx, actor.UserID); err != nil {
			return uuid.Nil, err
		}
	}

	var owner *uuid.UUID
	if in.TargetKind == TargetUser {
		id := in.TargetID
		owner = &id
	} else if s.owner != nil {
		id, found, err := s.owner(ctx, in.TargetKind, in.TargetID)
		if err != nil {
			return uuid.Nil, err
		}
		if found {
			owner = &id
		}
	}
	if owner != nil && *owner == actor.UserID {
		return uuid.Nil, ErrSelf
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return uuid.Nil, fmt.Errorf("reports: begin: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var id uuid.UUID
	err = tx.QueryRow(ctx, `
		INSERT INTO reports (reporter_id, target_kind, target_id, target_owner, reason, comment, counts_toward_hide)
		VALUES ($1, $2, $3, $4, $5, NULLIF($6, ''), $7)
		ON CONFLICT (reporter_id, target_kind, target_id) DO UPDATE
		  SET reason = EXCLUDED.reason, comment = EXCLUDED.comment,
		      counts_toward_hide = EXCLUDED.counts_toward_hide
		RETURNING id`,
		actor.UserID, in.TargetKind, in.TargetID, owner, in.Reason, comment, counts).Scan(&id)
	if err != nil {
		return uuid.Nil, fmt.Errorf("reports: create: %w", err)
	}

	if s.audit != nil {
		target := in.TargetID
		var session *uuid.UUID
		if actor.SessionID != uuid.Nil {
			session = &actor.SessionID
		}
		if err := s.audit.Record(ctx, tx, audit.Event{
			ActorID:    &actor.UserID,
			Action:     "report.create",
			TargetType: in.TargetKind,
			TargetID:   &target,
			IPHash:     actor.IPHash,
			SessionID:  session,
			RequestID:  actor.RequestID,
			Metadata:   map[string]any{"reason": in.Reason},
		}); err != nil {
			return uuid.Nil, err
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return uuid.Nil, fmt.Errorf("reports: commit: %w", err)
	}
	return id, nil
}

// ContentLoader returns what a reported thing says, when the server can read
// it at all. readable is false for a private message: it is encrypted, and the
// queue shows that rather than an empty box.
type ContentLoader func(ctx context.Context, kind string, id uuid.UUID) (content string, readable bool, err error)

// SetContentLoader wires the queue's content preview.
func (s *Service) SetContentLoader(f ContentLoader) { s.content = f }

// List returns the queue: reports of the given status, newest first.
func (s *Service) List(ctx context.Context, status string, limit int) ([]Report, error) {
	if status == "" {
		status = StatusOpen
	}
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	rows, err := s.pool.Query(ctx, `
		SELECT r.id, r.reporter_id, r.target_kind, r.target_id, r.target_owner,
		       r.reason, COALESCE(r.comment, ''), r.status, r.created_at,
		       (SELECT count(*) FROM reports x
		         WHERE x.target_kind = r.target_kind AND x.target_id = r.target_id AND x.status = 'open'),
		       (SELECT count(*) FROM reports x
		         WHERE x.target_owner IS NOT NULL AND x.target_owner = r.target_owner AND x.status = 'open')
		FROM reports r
		WHERE r.status = $1
		ORDER BY r.created_at DESC
		LIMIT $2`, status, limit)
	if err != nil {
		return nil, fmt.Errorf("reports: list: %w", err)
	}
	defer rows.Close()

	out := []Report{}
	for rows.Next() {
		var rep Report
		if err := rows.Scan(&rep.ID, &rep.ReporterID, &rep.TargetKind, &rep.TargetID, &rep.TargetOwner,
			&rep.Reason, &rep.Comment, &rep.Status, &rep.CreatedAt, &rep.SameTarget, &rep.AgainstOwner); err != nil {
			return nil, fmt.Errorf("reports: scan: %w", err)
		}
		out = append(out, rep)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if s.content != nil {
		for i := range out {
			content, readable, err := s.content(ctx, out[i].TargetKind, out[i].TargetID)
			if err != nil {
				return nil, err
			}
			out[i].Readable = readable
			if readable {
				text := content
				out[i].Content = &text
			}
		}
	}
	return out, nil
}

// Resolve closes a report. rejected marks it as nothing to act on; anything
// else counts as acted upon. The action itself (deleting a post, sanctioning
// an account) is taken through the existing moderation endpoints — this only
// records that the queue entry is done with.
func (s *Service) Resolve(ctx context.Context, actor ActorMeta, id uuid.UUID, rejected bool) error {
	status := StatusResolved
	if rejected {
		status = StatusRejected
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("reports: begin resolve: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var targetKind string
	var targetID uuid.UUID
	err = tx.QueryRow(ctx, `
		UPDATE reports SET status = $3, resolved_by = $2, resolved_at = now()
		WHERE id = $1 AND status = 'open'
		RETURNING target_kind, target_id`, id, actor.UserID, status).Scan(&targetKind, &targetID)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		return fmt.Errorf("reports: resolve: %w", err)
	}

	if s.audit != nil {
		target := targetID
		if err := s.audit.Record(ctx, tx, audit.Event{
			ActorID:    &actor.UserID,
			Action:     "report." + status,
			TargetType: targetKind,
			TargetID:   &target,
			IPHash:     actor.IPHash,
			RequestID:  actor.RequestID,
		}); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

// Counts is the summary the admin screen shows above the queue.
type Counts struct {
	Open     int `json:"open"`
	Resolved int `json:"resolved"`
	Rejected int `json:"rejected"`
	// Hidden: posts currently held back by reports alone.
	Hidden int `json:"hidden"`
}

// Summary counts the queue.
func (s *Service) Summary(ctx context.Context) (Counts, error) {
	var c Counts
	err := s.pool.QueryRow(ctx, `
		SELECT
			count(*) FILTER (WHERE status = 'open'),
			count(*) FILTER (WHERE status = 'resolved'),
			count(*) FILTER (WHERE status = 'rejected'),
			(SELECT count(*) FROM (
				SELECT target_id FROM reports
				WHERE target_kind = 'post' AND status = 'open'
				GROUP BY target_id HAVING count(*) >= $1
			) hidden)
		FROM reports`, AutoHideThreshold).Scan(&c.Open, &c.Resolved, &c.Rejected, &c.Hidden)
	if err != nil {
		return c, fmt.Errorf("reports: summary: %w", err)
	}
	return c, nil
}
