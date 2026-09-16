package moderation

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"kisy-backend/internal/access"
	"kisy-backend/internal/audit"
	"kisy-backend/internal/platform/db"
)

// Audit actions.
const (
	actionWarned    = "group.warned"
	actionMuted     = "group.muted"
	actionDeleted   = "group.moderation_deleted"
	actionRevoked   = "group.sanction_revoked"
	actionRestored  = "group.restored"
	actionPurged    = "group.purged"
	restoredNote    = "Сообщество восстановлено"
	warnsCappedNote = "Восстановление: счётчик предупреждений сброшен"
	replacedNote    = "Заменён новым мутом"
)

// Notice is one announcement to the people who run a group. Kept free of the
// notifications package so moderation does not depend on how it is delivered.
type Notice struct {
	Recipients []uuid.UUID
	Payload    map[string]any
	Text       string
	URL        string
}

// Notifier delivers a Notice (stored notification, live event, push).
type Notifier interface {
	Notify(ctx context.Context, n Notice) error
}

// RoleReader tells whether an actor may see a group's banner: its founder or a
// member of the editor tier. Satisfied by an adapter over the groups service,
// so visibility rules stay written once.
type RoleReader interface {
	// EditorTier returns ErrNotFound when the actor may not see the group at
	// all, and false when they see it but do not run it.
	EditorTier(ctx context.Context, groupID uuid.UUID, actor ActorMeta) (bool, error)
}

type Service struct {
	pool     *pgxpool.Pool
	repo     *Repository
	audit    audit.Recorder
	notifier Notifier
	roles    RoleReader
	// feedChanged recomputes the popularity ranking after a change that adds
	// or removes a community's posts from the feed.
	feedChanged func(ctx context.Context)
	// groupChanged tells a group's members to refetch it.
	groupChanged func(groupID uuid.UUID)
	now          func() time.Time
	log          *slog.Logger
}

func NewService(pool *pgxpool.Pool, repo *Repository, rec audit.Recorder, log *slog.Logger) *Service {
	return &Service{pool: pool, repo: repo, audit: rec, now: time.Now, log: log}
}

func (s *Service) SetNotifier(n Notifier)                      { s.notifier = n }
func (s *Service) SetRoleReader(r RoleReader)                  { s.roles = r }
func (s *Service) SetFeedChanged(fn func(ctx context.Context)) { s.feedChanged = fn }
func (s *Service) SetGroupChanged(fn func(groupID uuid.UUID))  { s.groupChanged = fn }
func (s *Service) SetClock(now func() time.Time)               { s.now = now }

// IssueInput is a sanction as the CEO asks for it.
type IssueInput struct {
	GroupID uuid.UUID
	Kind    string
	Reason  string
	// Duration is a key of MuteDurations; mutes only.
	Duration string
}

func cleanReason(raw string) (string, error) {
	reason := strings.TrimSpace(raw)
	if reason == "" {
		return "", ErrReasonRequired
	}
	if len([]rune(reason)) > maxReasonLength {
		return "", ErrReasonTooLong
	}
	return reason, nil
}

// Issue warns, mutes or deletes a group.
func (s *Service) Issue(ctx context.Context, in IssueInput, actor ActorMeta) (*Outcome, error) {
	if !access.IsCEO(actor.RoleLevel) {
		// The router already allows only the CEO; this keeps the rule true for
		// any other caller of the service.
		return nil, ErrForbidden
	}
	reason, err := cleanReason(in.Reason)
	if err != nil {
		return nil, err
	}
	var expires *time.Time
	switch in.Kind {
	case KindWarn, KindDelete:
	case KindMute:
		d, ok := MuteDurations[in.Duration]
		if !ok {
			return nil, ErrInvalidDuration
		}
		if d > 0 {
			t := s.now().UTC().Add(d)
			expires = &t
		}
	default:
		return nil, ErrInvalidKind
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("moderation: begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	g, err := s.repo.lockGroup(ctx, tx, in.GroupID)
	if err != nil {
		return nil, err
	}
	if g.DeletedAt != nil {
		return nil, ErrGroupDeleted
	}

	now := s.now().UTC()
	out := &Outcome{Sanction: Sanction{GroupID: g.ID, Kind: in.Kind, Reason: reason, IssuedBy: actor.UserID, ExpiresAt: expires}}

	if in.Kind == KindMute {
		// One live mute at a time: a new one replaces the old, so "muted until"
		// always has a single answer.
		if prev, err := s.repo.liveMute(ctx, tx, g.ID); err != nil {
			return nil, err
		} else if prev != nil {
			if _, err := s.repo.revoke(ctx, tx, prev.ID, actor.UserID, replacedNote, now); err != nil {
				return nil, err
			}
		}
	}
	if err := s.repo.insert(ctx, tx, &out.Sanction); err != nil {
		return nil, err
	}

	warns, err := s.repo.activeWarns(ctx, tx, g.ID)
	if err != nil {
		return nil, err
	}
	out.ActiveWarns = len(warns)

	action := map[string]string{KindWarn: actionWarned, KindMute: actionMuted, KindDelete: actionDeleted}[in.Kind]
	meta := map[string]any{"reason": reason, "sanctionId": out.Sanction.ID.String(), "activeWarns": out.ActiveWarns}
	if expires != nil {
		meta["expiresAt"] = expires
	}
	if err := s.record(ctx, tx, actor, action, g.ID, meta); err != nil {
		return nil, err
	}

	// Deletion: asked for, or the warning that reached the limit — which then
	// deletes exactly as a deletion by hand does, sanction row included.
	var autoDelete *Sanction
	if in.Kind == KindDelete || (in.Kind == KindWarn && out.ActiveWarns >= WarnLimit) {
		if in.Kind == KindWarn {
			autoDelete = &Sanction{
				GroupID:  g.ID,
				Kind:     KindDelete,
				Reason:   fmt.Sprintf("%d-е предупреждение: %s", WarnLimit, reason),
				IssuedBy: actor.UserID,
			}
			if err := s.repo.insert(ctx, tx, autoDelete); err != nil {
				return nil, err
			}
			if err := s.record(ctx, tx, actor, actionDeleted, g.ID, map[string]any{
				"reason": autoDelete.Reason, "sanctionId": autoDelete.ID.String(), "automatic": true,
			}); err != nil {
				return nil, err
			}
		}
		if err := s.repo.setDeleted(ctx, tx, g.ID, &actor.UserID, &now); err != nil {
			return nil, err
		}
		out.Deleted = true
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("moderation: commit: %w", err)
	}

	s.announce(ctx, g, in.Kind, out.Sanction, out.ActiveWarns)
	if autoDelete != nil {
		s.announce(ctx, g, KindDelete, *autoDelete, out.ActiveWarns)
	}
	if in.Kind != KindWarn || out.Deleted {
		s.touchFeed(ctx)
	}
	s.touchGroup(g.ID)
	return out, nil
}

// Revoke ends a live warning or mute. A revoked warning stops counting towards
// the limit; a revoked mute puts the community back in the feed at once.
func (s *Service) Revoke(ctx context.Context, sanctionID uuid.UUID, note string, actor ActorMeta) (*Sanction, error) {
	if !access.IsCEO(actor.RoleLevel) {
		return nil, ErrForbidden
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("moderation: begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	sanction, err := s.repo.get(ctx, tx, sanctionID)
	if err != nil {
		return nil, err
	}
	if _, err := s.repo.lockGroup(ctx, tx, sanction.GroupID); err != nil {
		return nil, err
	}
	// Deletion is undone by restoring, which also settles the warnings.
	if sanction.Kind == KindDelete || !sanction.Active(s.now()) {
		return nil, ErrNotRevocable
	}
	revoked, err := s.repo.revoke(ctx, tx, sanction.ID, actor.UserID, note, s.now().UTC())
	if err != nil {
		return nil, err
	}
	if !revoked {
		return nil, ErrNotRevocable
	}
	if err := s.record(ctx, tx, actor, actionRevoked, sanction.GroupID, map[string]any{
		"sanctionId": sanction.ID.String(), "kind": sanction.Kind, "note": note,
	}); err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("moderation: commit: %w", err)
	}
	if sanction.Kind == KindMute {
		s.touchFeed(ctx)
	}
	s.touchGroup(sanction.GroupID)
	return s.repo.get(ctx, s.pool, sanction.ID)
}

// Restore brings a deleted group back within RestoreWindow, with everything it
// had. Its live warnings are cut to WarnLimit-1, so the next one deletes it
// again; fewer than that are left as they are — a restore does not invent
// warnings nobody issued.
func (s *Service) Restore(ctx context.Context, groupID uuid.UUID, actor ActorMeta) error {
	if !access.IsCEO(actor.RoleLevel) {
		return ErrForbidden
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("moderation: begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	g, err := s.repo.lockGroup(ctx, tx, groupID)
	if err != nil {
		return err
	}
	if g.DeletedAt == nil {
		return ErrNotDeleted
	}
	now := s.now().UTC()
	if now.After(g.DeletedAt.Add(RestoreWindow)) {
		return ErrRestoreExpired
	}

	if err := s.repo.setDeleted(ctx, tx, g.ID, nil, nil); err != nil {
		return err
	}
	if err := s.repo.revokeLiveDeletions(ctx, tx, g.ID, actor.UserID, restoredNote, now); err != nil {
		return err
	}
	warns, err := s.repo.activeWarns(ctx, tx, g.ID)
	if err != nil {
		return err
	}
	capped := 0
	for i := 0; i < len(warns)-(WarnLimit-1); i++ {
		if _, err := s.repo.revoke(ctx, tx, warns[i].ID, actor.UserID, warnsCappedNote, now); err != nil {
			return err
		}
		capped++
	}
	if err := s.record(ctx, tx, actor, actionRestored, g.ID, map[string]any{
		"deletedAt": g.DeletedAt, "warnsRevoked": capped,
	}); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("moderation: commit: %w", err)
	}

	s.announceRestore(ctx, g)
	s.touchFeed(ctx)
	s.touchGroup(g.ID)
	return nil
}

// Purge removes, for good, every group whose restore window closed before now.
func (s *Service) Purge(ctx context.Context, now time.Time) (int, error) {
	expired, err := s.repo.expired(ctx, s.pool, now.Add(-RestoreWindow))
	if err != nil {
		return 0, err
	}
	purged := 0
	for _, g := range expired {
		tx, err := s.pool.Begin(ctx)
		if err != nil {
			return purged, fmt.Errorf("moderation: begin tx: %w", err)
		}
		done, err := s.repo.purge(ctx, tx, g.ID)
		if err == nil && done {
			gid := g.ID
			err = s.audit.Record(ctx, tx, audit.Event{
				Action:     actionPurged,
				TargetType: "group",
				TargetID:   &gid,
				Metadata:   map[string]any{"name": g.Name, "kind": g.Kind, "deletedAt": g.DeletedAt, "founder": g.CreatedBy.String()},
			})
		}
		if err != nil {
			_ = tx.Rollback(ctx)
			return purged, err
		}
		if err := tx.Commit(ctx); err != nil {
			return purged, fmt.Errorf("moderation: commit purge: %w", err)
		}
		if done {
			purged++
		}
	}
	return purged, nil
}

// StartPurgeWorker runs Purge now and then every interval until ctx ends.
func (s *Service) StartPurgeWorker(ctx context.Context, interval time.Duration) {
	go func() {
		run := func() {
			n, err := s.Purge(ctx, s.now())
			if err != nil && ctx.Err() == nil {
				s.log.Warn("moderation: purge failed", "error", err)
			} else if n > 0 {
				s.log.Info("moderation: purged deleted groups", "count", n)
			}
		}
		run()
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				run()
			}
		}
	}()
}

// ListGroups is the CEO's communities list with each group's live sanctions.
func (s *Service) ListGroups(ctx context.Context, query string) ([]GroupSummary, error) {
	return s.repo.listGroups(ctx, s.pool, query, 100)
}

// ListDeleted is the CEO's list of groups waiting in their restore window.
func (s *Service) ListDeleted(ctx context.Context) ([]DeletedGroup, error) {
	return s.repo.listDeleted(ctx, s.pool)
}

// History is every sanction a group ever had, newest first.
func (s *Service) History(ctx context.Context, groupID uuid.UUID) ([]Sanction, error) {
	if _, err := s.repo.getGroup(ctx, s.pool, groupID); err != nil {
		return nil, err
	}
	return s.repo.history(ctx, s.pool, groupID)
}

// ActiveFor is the banner a group's own editors see: its live warnings and mute.
func (s *Service) ActiveFor(ctx context.Context, groupID uuid.UUID, actor ActorMeta) (*ActiveSanctions, error) {
	if !access.IsCEO(actor.RoleLevel) {
		if s.roles == nil {
			return nil, ErrForbidden
		}
		ok, err := s.roles.EditorTier(ctx, groupID, actor)
		if err != nil {
			return nil, err
		}
		if !ok {
			return nil, ErrForbidden
		}
	}
	warns, err := s.repo.activeWarns(ctx, s.pool, groupID)
	if err != nil {
		return nil, err
	}
	mute, err := s.repo.liveMute(ctx, s.pool, groupID)
	if err != nil {
		return nil, err
	}
	return &ActiveSanctions{Warns: warns, WarnLimit: WarnLimit, Mute: mute}, nil
}

// record writes one audit event inside the action's transaction: an action
// that is not in the audit log did not happen.
func (s *Service) record(ctx context.Context, q db.DBTX, actor ActorMeta, action string, groupID uuid.UUID, meta map[string]any) error {
	gid := groupID
	return s.audit.Record(ctx, q, audit.Event{
		ActorID:    &actor.UserID,
		Action:     action,
		TargetType: "group",
		TargetID:   &gid,
		IPHash:     actor.IPHash,
		SessionID:  &actor.SessionID,
		RequestID:  actor.RequestID,
		Metadata:   meta,
	})
}
