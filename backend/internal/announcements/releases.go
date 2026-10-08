package announcements

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"kisy-backend/internal/access"
	"kisy-backend/internal/audit"
	"kisy-backend/internal/i18n"
)

// A new version of the app, announced by the CEO to every active account
// ("New Update" in the admin panel): what it is called, what changed and
// where to get it. Same delivery as an announcement — the list, live, push.

// ReleaseNotificationType is the type of the notifications rows it writes.
const ReleaseNotificationType = "app_release"

// ActionReleaseAnnounced is its audit action.
const ActionReleaseAnnounced = "release.announced"

const (
	MaxReleaseVersion = 32
	MaxReleaseNotes   = 4000
)

// ErrNotCEO: only the CEO announces versions.
var ErrNotCEO = errors.New("announcements: only the CEO announces a new version")

// ReleaseInput is a new version to announce.
type ReleaseInput struct {
	Version     string
	Notes       string
	DownloadURL string
}

// Release is one announced version.
type Release struct {
	ID             uuid.UUID `json:"id"`
	Version        string    `json:"version"`
	Notes          string    `json:"notes"`
	DownloadURL    *string   `json:"downloadUrl,omitempty"`
	RecipientCount int       `json:"recipientCount"`
	CreatedAt      time.Time `json:"createdAt"`
}

func normalizeRelease(in ReleaseInput) (ReleaseInput, error) {
	in.Version = strings.TrimSpace(in.Version)
	in.Notes = strings.TrimSpace(in.Notes)
	in.DownloadURL = strings.TrimSpace(in.DownloadURL)
	if in.Version == "" || in.Notes == "" ||
		utf8.RuneCountInString(in.Version) > MaxReleaseVersion || utf8.RuneCountInString(in.Notes) > MaxReleaseNotes {
		return in, ErrValidation
	}
	if in.DownloadURL != "" {
		u, err := url.Parse(in.DownloadURL)
		if err != nil || u.Scheme != "https" || u.Host == "" {
			return in, ErrValidation
		}
	}
	return in, nil
}

// SendRelease announces a version to every active account but the CEO.
func (s *Service) SendRelease(ctx context.Context, actor ActorMeta, in ReleaseInput) (*Release, error) {
	in, err := normalizeRelease(in)
	if err != nil {
		return nil, err
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	level, _, err := authorLevel(ctx, tx, actor.UserID)
	if err != nil {
		return nil, err
	}
	if !access.IsCEO(level) {
		return nil, ErrNotCEO
	}

	r := &Release{Version: in.Version, Notes: in.Notes}
	var link *string
	if in.DownloadURL != "" {
		link = &in.DownloadURL
		r.DownloadURL = link
	}
	if err := tx.QueryRow(ctx, `
		INSERT INTO app_releases (version, notes, download_url, created_by)
		VALUES ($1, $2, $3, $4) RETURNING id, created_at`,
		in.Version, in.Notes, link, actor.UserID).Scan(&r.ID, &r.CreatedAt); err != nil {
		return nil, fmt.Errorf("announcements: insert release: %w", err)
	}

	payload := map[string]any{"releaseId": r.ID, "version": r.Version, "notes": r.Notes}
	if link != nil {
		payload["downloadUrl"] = *link
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("announcements: encode release: %w", err)
	}
	rows, err := tx.Query(ctx, `
		INSERT INTO notifications (user_id, type, payload)
		SELECT id, $1, $2 FROM users WHERE is_active AND id <> $3
		RETURNING user_id`, ReleaseNotificationType, raw, actor.UserID)
	if err != nil {
		return nil, fmt.Errorf("announcements: release fan out: %w", err)
	}
	recipients, err := pgx.CollectRows(rows, pgx.RowTo[uuid.UUID])
	if err != nil {
		return nil, fmt.Errorf("announcements: release fan out: %w", err)
	}
	r.RecipientCount = len(recipients)
	if _, err := tx.Exec(ctx, `UPDATE app_releases SET recipient_count = $2 WHERE id = $1`, r.ID, r.RecipientCount); err != nil {
		return nil, fmt.Errorf("announcements: release count: %w", err)
	}
	if err := s.audit.Record(ctx, tx, audit.Event{
		ActorID: &actor.UserID, Action: ActionReleaseAnnounced, TargetType: "app_release", TargetID: &r.ID,
		IPHash: actor.IPHash, SessionID: &actor.SessionID, RequestID: actor.RequestID,
		Metadata: map[string]any{"version": r.Version, "recipients": r.RecipientCount},
	}); err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	// The push opens the app, where the notification carries the link: a tap
	// is routed inside the app, and an outside address would go nowhere.
	s.fanOut(recipients, ReleaseNotificationType, payload, i18n.M("releases.pushTitle", r.Version), i18n.Raw(excerpt(r.Notes, 160)), "/")
	return r, nil
}

// Releases lists announced versions, newest first.
func (s *Service) Releases(ctx context.Context, limit int) ([]Release, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	rows, err := s.pool.Query(ctx, `
		SELECT id, version, notes, download_url, recipient_count, created_at
		FROM app_releases ORDER BY created_at DESC LIMIT $1`, limit)
	if err != nil {
		return nil, fmt.Errorf("announcements: releases: %w", err)
	}
	out, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (Release, error) {
		var r Release
		return r, row.Scan(&r.ID, &r.Version, &r.Notes, &r.DownloadURL, &r.RecipientCount, &r.CreatedAt)
	})
	if err != nil {
		return nil, fmt.Errorf("announcements: releases: %w", err)
	}
	if out == nil {
		out = []Release{}
	}
	return out, nil
}

func excerpt(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return strings.TrimSpace(string(r[:n])) + "…"
}
