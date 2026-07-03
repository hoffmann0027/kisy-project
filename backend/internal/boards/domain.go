// Package boards implements Trello-style task boards attached to groups.
// One board per group: the group founder owns the board structure
// (columns); any group member manages cards.
package boards

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

var (
	ErrNotFound      = errors.New("boards: not found")
	ErrForbidden     = errors.New("boards: not permitted")
	ErrBoardExists   = errors.New("boards: board already exists for this group")
	ErrInvalidInput  = errors.New("boards: invalid input")
	ErrColumnMissing = errors.New("boards: column does not belong to this board")
)

type Board struct {
	ID        uuid.UUID
	GroupID   uuid.UUID
	Title     string
	CreatedBy uuid.UUID
	CreatedAt time.Time
}

type Column struct {
	ID       uuid.UUID
	BoardID  uuid.UUID
	Title    string
	Position int
}

type Card struct {
	ID          uuid.UUID
	BoardID     uuid.UUID
	ColumnID    uuid.UUID
	Title       string
	Description *string
	Position    int
	AssigneeID  *uuid.UUID
	Label       *string
	DueDate     *time.Time
	CreatedBy   uuid.UUID
	CreatedAt   time.Time
}

// --- DTOs ---

type CardDTO struct {
	ID          uuid.UUID  `json:"id"`
	ColumnID    uuid.UUID  `json:"columnId"`
	Title       string     `json:"title"`
	Description *string    `json:"description"`
	Position    int        `json:"position"`
	AssigneeID  *uuid.UUID `json:"assigneeId"`
	Label       *string    `json:"label"`
	DueDate     *time.Time `json:"dueDate"`
	CreatedBy   uuid.UUID  `json:"createdBy"`
	CreatedAt   time.Time  `json:"createdAt"`
}

type ColumnDTO struct {
	ID       uuid.UUID `json:"id"`
	Title    string    `json:"title"`
	Position int       `json:"position"`
	Cards    []CardDTO `json:"cards"`
}

type BoardDTO struct {
	ID        uuid.UUID   `json:"id"`
	GroupID   uuid.UUID   `json:"groupId"`
	Title     string      `json:"title"`
	CreatedBy uuid.UUID   `json:"createdBy"`
	Columns   []ColumnDTO `json:"columns"`
}

func (c *Card) ToDTO() CardDTO {
	return CardDTO{
		ID:          c.ID,
		ColumnID:    c.ColumnID,
		Title:       c.Title,
		Description: c.Description,
		Position:    c.Position,
		AssigneeID:  c.AssigneeID,
		Label:       c.Label,
		DueDate:     c.DueDate,
		CreatedBy:   c.CreatedBy,
		CreatedAt:   c.CreatedAt,
	}
}
