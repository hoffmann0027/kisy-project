// Package rating implements the "Рейтинг" project board: a three-column
// kanban (projects → in progress → done) with a per-project net-profit ledger
// and analytics. All authority (CEO creates projects/tasks; a user may assign
// only themselves; only the assignee moves progress; assignee or CEO record
// finances) is enforced in the service, never merely hidden in the UI.
package rating

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

var (
	ErrNotFound       = errors.New("rating: not found")
	ErrForbidden      = errors.New("rating: not permitted")
	ErrValidation     = errors.New("rating: invalid input")
	ErrAlreadyClaimed = errors.New("rating: task already has an assignee")
)

// Task lifecycle columns.
const (
	StatusBacklog    = "backlog"
	StatusInProgress = "in_progress"
	StatusDone       = "done"
)

var validDifficulty = map[string]bool{"easy": true, "medium": true, "hard": true}

// Actor identifies the acting user for authorization.
type Actor struct {
	UserID    uuid.UUID
	RoleLevel int
}

func (a Actor) isCEO() bool { return a.RoleLevel == 1 }

// Assignee is the public identity of a task's executor.
type Assignee struct {
	ID          uuid.UUID `json:"id"`
	DisplayName string    `json:"displayName"`
	AvatarURL   *string   `json:"avatarUrl"`
}

// TaskDTO is one task card. TotalProfitKopecks is the sum of the task's
// ledger entries (income − expense), used to sort the "done" column.
type TaskDTO struct {
	ID                 uuid.UUID `json:"id"`
	ProjectID          uuid.UUID `json:"projectId"`
	ProjectTitle       string    `json:"projectTitle"`
	Title              string    `json:"title"`
	Assignee           *Assignee `json:"assignee"`
	Progress           int       `json:"progress"`
	Status             string    `json:"status"`
	TotalProfitKopecks int64     `json:"totalProfitKopecks"`
	CreatedAt          time.Time `json:"createdAt"`
}

// ProjectDTO is a backlog project card with its tasks embedded.
type ProjectDTO struct {
	ID                 uuid.UUID `json:"id"`
	Title              string    `json:"title"`
	Description        *string   `json:"description"`
	Difficulty         string    `json:"difficulty"`
	CreatedBy          uuid.UUID `json:"createdBy"`
	TotalProfitKopecks int64     `json:"totalProfitKopecks"`
	Tasks              []TaskDTO `json:"tasks"`
	CreatedAt          time.Time `json:"createdAt"`
}

// FinanceEntryDTO is one ledger record.
type FinanceEntryDTO struct {
	ID             uuid.UUID `json:"id"`
	IncomeKopecks  int64     `json:"incomeKopecks"`
	ExpenseKopecks int64     `json:"expenseKopecks"`
	ProfitKopecks  int64     `json:"profitKopecks"`
	Note           *string   `json:"note"`
	CreatedBy      uuid.UUID `json:"createdBy"`
	CreatedAt      time.Time `json:"createdAt"`
}

// AnalyticsDTO powers the two charts: per-project profit share (pie) and
// total monthly profit across all projects (line).
type AnalyticsDTO struct {
	PerProject []ProjectProfit `json:"perProject"`
	Monthly    []MonthlyProfit `json:"monthly"`
}

type ProjectProfit struct {
	ProjectID     uuid.UUID `json:"projectId"`
	Title         string    `json:"title"`
	ProfitKopecks int64     `json:"profitKopecks"`
}

type MonthlyProfit struct {
	Month         string `json:"month"` // "YYYY-MM"
	ProfitKopecks int64  `json:"profitKopecks"`
}
