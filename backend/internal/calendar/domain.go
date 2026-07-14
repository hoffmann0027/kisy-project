// Package calendar owns per-group calendars: one-off events visible to every
// group member, plus a read-only view of board cards that carry a due date.
package calendar

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
)

var (
	ErrNotFound     = errors.New("calendar: not found")
	ErrForbidden    = errors.New("calendar: not permitted")
	ErrValidation   = errors.New("calendar: invalid input")
	ErrBadColor     = errors.New("calendar: color not in palette")
	ErrBadTimeRange = errors.New("calendar: endsAt must be >= startsAt")
)

// Palette is the fixed set of event colours (kept in sync with the DB CHECK
// and the frontend picker).
var Palette = map[string]bool{
	"blue": true, "green": true, "red": true, "orange": true,
	"purple": true, "teal": true, "pink": true, "gray": true,
}

const maxTitleLen = 200

// Event mirrors a calendar_events row.
type Event struct {
	ID        uuid.UUID
	GroupID   uuid.UUID
	Title     string
	StartsAt  time.Time
	EndsAt    *time.Time
	Color     string
	CreatedBy uuid.UUID
	CreatedAt time.Time
	UpdatedAt time.Time
}

// EventDTO is the API representation of a calendar event.
type EventDTO struct {
	Kind      string     `json:"kind"` // always "event"
	ID        uuid.UUID  `json:"id"`
	GroupID   uuid.UUID  `json:"groupId"`
	Title     string     `json:"title"`
	StartsAt  time.Time  `json:"startsAt"`
	EndsAt    *time.Time `json:"endsAt"`
	Color     string     `json:"color"`
	CreatedBy uuid.UUID  `json:"createdBy"`
}

func (e *Event) ToDTO() EventDTO {
	return EventDTO{
		Kind: "event", ID: e.ID, GroupID: e.GroupID, Title: e.Title,
		StartsAt: e.StartsAt, EndsAt: e.EndsAt, Color: e.Color, CreatedBy: e.CreatedBy,
	}
}

// CardRef is a board card surfaced in the calendar by its due date (read-only;
// clicking it opens the card on the board).
type CardRef struct {
	CardID   uuid.UUID `json:"cardId"`
	Title    string    `json:"title"`
	DueDate  time.Time `json:"dueDate"`
	ColumnID uuid.UUID `json:"columnId"`
}

// Actor is the acting user.
type Actor struct {
	UserID    uuid.UUID
	RoleLevel int
	SessionID uuid.UUID
	IPHash    string
	RequestID string
}

// Access injects group visibility/role checks (avoids a calendar→groups
// import cycle). EnsureMember returns groups.ErrNotFound-equivalent (mapped to
// calendar.ErrNotFound) for a hidden/inaccessible group and ErrForbidden for a
// non-member. IsFounder reports group ownership.
type Access struct {
	EnsureMember func(ctx context.Context, groupID, actorID uuid.UUID, actorLevel int) error
	IsFounder    func(ctx context.Context, groupID, userID uuid.UUID) (bool, error)
}

// CardLoader returns a group's board cards whose due date falls within
// [from, to). Wired to the boards module.
type CardLoader func(ctx context.Context, groupID uuid.UUID, from, to time.Time) ([]CardRef, error)

// validateColor / validateTitle guard incoming values.
func validColor(c string) bool { return Palette[c] }
