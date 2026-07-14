package calendar

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"kisy-backend/internal/access"
	"kisy-backend/internal/audit"
	"kisy-backend/internal/platform/db"
)

const (
	actionCalendarModerated = "calendar.event_moderated"
)

// ChangePublisher notifies a group's members that its calendar changed.
type ChangePublisher func(groupID uuid.UUID)

type Service struct {
	pool    *pgxpool.Pool
	repo    Repository
	audit   audit.Recorder
	access  Access
	cards   CardLoader
	changed ChangePublisher
}

func NewService(pool *pgxpool.Pool, repo Repository, rec audit.Recorder, acc Access) *Service {
	return &Service{pool: pool, repo: repo, audit: rec, access: acc}
}

func (s *Service) SetCardLoader(l CardLoader)           { s.cards = l }
func (s *Service) SetChangePublisher(p ChangePublisher) { s.changed = p }

func (s *Service) broadcast(groupID uuid.UUID) {
	if s.changed != nil {
		s.changed(groupID)
	}
}

// MonthView bundles a group's events and due-dated board cards for an interval.
type MonthView struct {
	Events []EventDTO `json:"events"`
	Cards  []CardRef  `json:"cards"`
}

// List returns the group's events and board-card references overlapping
// [from, to). Requires membership; a hidden group is ErrNotFound.
func (s *Service) List(ctx context.Context, groupID uuid.UUID, from, to time.Time, actor Actor) (MonthView, error) {
	if err := s.access.EnsureMember(ctx, groupID, actor.UserID, actor.RoleLevel); err != nil {
		return MonthView{}, err
	}
	events, err := s.repo.ListByGroupInterval(ctx, s.pool, groupID, from, to)
	if err != nil {
		return MonthView{}, err
	}
	view := MonthView{Events: make([]EventDTO, 0, len(events)), Cards: []CardRef{}}
	for i := range events {
		view.Events = append(view.Events, events[i].ToDTO())
	}
	if s.cards != nil {
		cards, err := s.cards(ctx, groupID, from, to)
		if err != nil {
			return MonthView{}, err
		}
		if cards != nil {
			view.Cards = cards
		}
	}
	return view, nil
}

// Input is a validated create/update payload.
type Input struct {
	Title    string
	StartsAt time.Time
	EndsAt   *time.Time
	Color    string
}

func (in Input) validate() error {
	if strings.TrimSpace(in.Title) == "" || len(in.Title) > maxTitleLen {
		return ErrValidation
	}
	if !validColor(in.Color) {
		return ErrBadColor
	}
	if in.EndsAt != nil && in.EndsAt.Before(in.StartsAt) {
		return ErrBadTimeRange
	}
	return nil
}

// Create adds an event. Any group member may create one.
func (s *Service) Create(ctx context.Context, groupID uuid.UUID, in Input, actor Actor) (*Event, error) {
	if err := s.access.EnsureMember(ctx, groupID, actor.UserID, actor.RoleLevel); err != nil {
		return nil, err
	}
	if err := in.validate(); err != nil {
		return nil, err
	}
	e := &Event{
		GroupID: groupID, Title: strings.TrimSpace(in.Title), StartsAt: in.StartsAt,
		EndsAt: in.EndsAt, Color: in.Color, CreatedBy: actor.UserID,
	}
	if err := s.repo.Create(ctx, s.pool, e); err != nil {
		return nil, err
	}
	s.broadcast(groupID)
	return e, nil
}

// canModerate reports whether the actor may edit/delete the event: its author,
// the group founder/owner, or the CEO.
func (s *Service) canModerate(ctx context.Context, e *Event, actor Actor) (bool, error) {
	if e.CreatedBy == actor.UserID || access.IsCEO(actor.RoleLevel) {
		return true, nil
	}
	return s.access.IsFounder(ctx, e.GroupID, actor.UserID)
}

// loadAccessible fetches an event only if the actor may see its group.
func (s *Service) loadAccessible(ctx context.Context, eventID uuid.UUID, actor Actor) (*Event, error) {
	e, err := s.repo.GetByID(ctx, s.pool, eventID)
	if err != nil {
		return nil, err // ErrNotFound propagates
	}
	if err := s.access.EnsureMember(ctx, e.GroupID, actor.UserID, actor.RoleLevel); err != nil {
		return nil, err
	}
	return e, nil
}

// Update edits an event. The author, group founder/owner or CEO may do it.
func (s *Service) Update(ctx context.Context, eventID uuid.UUID, in Input, actor Actor) (*Event, error) {
	e, err := s.loadAccessible(ctx, eventID, actor)
	if err != nil {
		return nil, err
	}
	ok, err := s.canModerate(ctx, e, actor)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, ErrForbidden
	}
	if err := in.validate(); err != nil {
		return nil, err
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("calendar: begin tx: %w", err)
	}
	defer tx.Rollback(ctx)
	if err := s.repo.Update(ctx, tx, eventID, strings.TrimSpace(in.Title), in.StartsAt, in.EndsAt, in.Color); err != nil {
		return nil, err
	}
	if e.CreatedBy != actor.UserID { // privileged moderation of another user's event
		if err := s.recordModeration(ctx, tx, e, actor, "update"); err != nil {
			return nil, err
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("calendar: commit: %w", err)
	}
	s.broadcast(e.GroupID)
	return s.repo.GetByID(ctx, s.pool, eventID)
}

// Delete removes an event. The author, group founder/owner or CEO may do it.
func (s *Service) Delete(ctx context.Context, eventID uuid.UUID, actor Actor) error {
	e, err := s.loadAccessible(ctx, eventID, actor)
	if err != nil {
		return err
	}
	ok, err := s.canModerate(ctx, e, actor)
	if err != nil {
		return err
	}
	if !ok {
		return ErrForbidden
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("calendar: begin tx: %w", err)
	}
	defer tx.Rollback(ctx)
	if err := s.repo.Delete(ctx, tx, eventID); err != nil {
		return err
	}
	if e.CreatedBy != actor.UserID {
		if err := s.recordModeration(ctx, tx, e, actor, "delete"); err != nil {
			return err
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("calendar: commit: %w", err)
	}
	s.broadcast(e.GroupID)
	return nil
}

func (s *Service) recordModeration(ctx context.Context, tx db.DBTX, e *Event, actor Actor, op string) error {
	id := e.ID
	return s.audit.Record(ctx, tx, audit.Event{
		ActorID:    &actor.UserID,
		Action:     actionCalendarModerated,
		TargetType: "calendar_event",
		TargetID:   &id,
		IPHash:     actor.IPHash,
		SessionID:  &actor.SessionID,
		RequestID:  actor.RequestID,
		Metadata:   map[string]any{"op": op, "author": e.CreatedBy.String(), "groupId": e.GroupID.String()},
	})
}
