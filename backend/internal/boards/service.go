package boards

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"kisy-backend/internal/platform/db"
)

// Access injects the group-membership and founder checks so the boards
// package does not import groups directly.
type Access struct {
	// EnsureActorMember returns nil if the actor may view/use the group's
	// board (member + clearance), else a non-nil error.
	EnsureActorMember func(ctx context.Context, groupID, actorID uuid.UUID, actorLevel int) error
	IsFounder         func(ctx context.Context, groupID, actorID uuid.UUID) (bool, error)
	IsMember          func(ctx context.Context, groupID, userID uuid.UUID) (bool, error)
}

// Publisher broadcasts board changes to the group's members over WebSocket.
type Publisher interface {
	PublishBoardChanged(groupID uuid.UUID)
}

// Default columns seeded when a board is created.
var defaultColumns = []string{"К выполнению", "В работе", "Готово"}

// Actor identifies the acting user.
type Actor struct {
	UserID    uuid.UUID
	RoleLevel int
}

type Service struct {
	pool   *pgxpool.Pool
	repo   Repository
	access Access
	pub    Publisher
}

func NewService(pool *pgxpool.Pool, repo Repository, access Access) *Service {
	return &Service{pool: pool, repo: repo, access: access}
}

func (s *Service) SetPublisher(p Publisher) { s.pub = p }

// --- board ---

// Get returns the group's board with columns and cards, if the actor is a
// member. ErrNotFound means the board has not been created yet.
func (s *Service) Get(ctx context.Context, groupID uuid.UUID, actor Actor) (*BoardDTO, error) {
	if err := s.access.EnsureActorMember(ctx, groupID, actor.UserID, actor.RoleLevel); err != nil {
		return nil, ErrForbidden
	}

	board, err := s.repo.GetBoardByGroup(ctx, s.pool, groupID)
	if err != nil {
		return nil, err // ErrNotFound propagates
	}
	return s.assemble(ctx, board)
}

func (s *Service) assemble(ctx context.Context, board *Board) (*BoardDTO, error) {
	columns, err := s.repo.ListColumns(ctx, s.pool, board.ID)
	if err != nil {
		return nil, err
	}
	cards, err := s.repo.ListCards(ctx, s.pool, board.ID)
	if err != nil {
		return nil, err
	}

	byColumn := make(map[uuid.UUID][]CardDTO)
	for i := range cards {
		byColumn[cards[i].ColumnID] = append(byColumn[cards[i].ColumnID], cards[i].ToDTO())
	}

	dto := &BoardDTO{ID: board.ID, GroupID: board.GroupID, Title: board.Title, CreatedBy: board.CreatedBy}
	for i := range columns {
		col := ColumnDTO{ID: columns[i].ID, Title: columns[i].Title, Position: columns[i].Position, Cards: byColumn[columns[i].ID]}
		if col.Cards == nil {
			col.Cards = []CardDTO{}
		}
		dto.Columns = append(dto.Columns, col)
	}
	if dto.Columns == nil {
		dto.Columns = []ColumnDTO{}
	}
	return dto, nil
}

// Create makes a board (founder only) with the default columns.
func (s *Service) Create(ctx context.Context, groupID uuid.UUID, title string, actor Actor) (*BoardDTO, error) {
	if err := s.requireFounder(ctx, groupID, actor); err != nil {
		return nil, err
	}
	if _, err := s.repo.GetBoardByGroup(ctx, s.pool, groupID); err == nil {
		return nil, ErrBoardExists
	}
	title = strings.TrimSpace(title)
	if title == "" {
		title = "Доска задач"
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("boards: begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	board := &Board{GroupID: groupID, Title: title, CreatedBy: actor.UserID}
	if err := s.repo.CreateBoard(ctx, tx, board); err != nil {
		return nil, err
	}
	for i, name := range defaultColumns {
		if err := s.repo.CreateColumn(ctx, tx, &Column{BoardID: board.ID, Title: name, Position: i}); err != nil {
			return nil, err
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("boards: commit: %w", err)
	}

	s.notify(board.GroupID)
	return s.assemble(ctx, board)
}

// --- columns (founder) ---

func (s *Service) AddColumn(ctx context.Context, boardID uuid.UUID, title string, actor Actor) error {
	board, err := s.boardForFounder(ctx, boardID, actor)
	if err != nil {
		return err
	}
	title = strings.TrimSpace(title)
	if title == "" {
		return ErrInvalidInput
	}
	pos, err := s.repo.MaxColumnPosition(ctx, s.pool, boardID)
	if err != nil {
		return err
	}
	if err := s.repo.CreateColumn(ctx, s.pool, &Column{BoardID: boardID, Title: title, Position: pos + 1}); err != nil {
		return err
	}
	s.notify(board.GroupID)
	return nil
}

func (s *Service) RenameColumn(ctx context.Context, columnID uuid.UUID, title string, actor Actor) error {
	col, err := s.repo.GetColumn(ctx, s.pool, columnID)
	if err != nil {
		return err
	}
	board, err := s.boardForFounder(ctx, col.BoardID, actor)
	if err != nil {
		return err
	}
	title = strings.TrimSpace(title)
	if title == "" {
		return ErrInvalidInput
	}
	if err := s.repo.RenameColumn(ctx, s.pool, columnID, title); err != nil {
		return err
	}
	s.notify(board.GroupID)
	return nil
}

func (s *Service) DeleteColumn(ctx context.Context, columnID uuid.UUID, actor Actor) error {
	col, err := s.repo.GetColumn(ctx, s.pool, columnID)
	if err != nil {
		return err
	}
	board, err := s.boardForFounder(ctx, col.BoardID, actor)
	if err != nil {
		return err
	}
	if err := s.repo.DeleteColumn(ctx, s.pool, columnID); err != nil {
		return err
	}
	s.notify(board.GroupID)
	return nil
}

// --- cards (any member) ---

// CardInput carries the mutable fields of a card.
type CardInput struct {
	Title       string
	Description *string
	AssigneeID  *uuid.UUID
	Label       *string
	DueDate     *time.Time
}

func (s *Service) CreateCard(ctx context.Context, columnID uuid.UUID, in CardInput, actor Actor) (*CardDTO, error) {
	col, err := s.repo.GetColumn(ctx, s.pool, columnID)
	if err != nil {
		return nil, err
	}
	board, err := s.repo.GetBoardByID(ctx, s.pool, col.BoardID)
	if err != nil {
		return nil, err
	}
	if err := s.requireMember(ctx, board.GroupID, actor); err != nil {
		return nil, err
	}

	title := strings.TrimSpace(in.Title)
	if title == "" {
		return nil, ErrInvalidInput
	}
	if err := s.validateAssignee(ctx, board.GroupID, in.AssigneeID); err != nil {
		return nil, err
	}

	pos, err := s.repo.MaxCardPosition(ctx, s.pool, columnID)
	if err != nil {
		return nil, err
	}
	card := &Card{
		BoardID: board.ID, ColumnID: columnID, Title: title, Description: in.Description,
		Position: pos + 1, AssigneeID: in.AssigneeID, Label: in.Label, DueDate: in.DueDate,
		CreatedBy: actor.UserID,
	}
	if err := s.repo.CreateCard(ctx, s.pool, card); err != nil {
		return nil, err
	}
	s.notify(board.GroupID)
	dto := card.ToDTO()
	return &dto, nil
}

func (s *Service) UpdateCard(ctx context.Context, cardID uuid.UUID, in CardInput, actor Actor) error {
	card, board, err := s.cardWithBoard(ctx, cardID)
	if err != nil {
		return err
	}
	if err := s.requireMember(ctx, board.GroupID, actor); err != nil {
		return err
	}
	title := strings.TrimSpace(in.Title)
	if title == "" {
		return ErrInvalidInput
	}
	if err := s.validateAssignee(ctx, board.GroupID, in.AssigneeID); err != nil {
		return err
	}

	card.Title = title
	card.Description = in.Description
	card.AssigneeID = in.AssigneeID
	card.Label = in.Label
	card.DueDate = in.DueDate
	if err := s.repo.UpdateCard(ctx, s.pool, card); err != nil {
		return err
	}
	s.notify(board.GroupID)
	return nil
}

func (s *Service) DeleteCard(ctx context.Context, cardID uuid.UUID, actor Actor) error {
	card, board, err := s.cardWithBoard(ctx, cardID)
	if err != nil {
		return err
	}
	if err := s.requireMember(ctx, board.GroupID, actor); err != nil {
		return err
	}
	// Only the card's creator or the group founder may delete it.
	if card.CreatedBy != actor.UserID {
		founder, err := s.access.IsFounder(ctx, board.GroupID, actor.UserID)
		if err != nil {
			return err
		}
		if !founder {
			return ErrForbidden
		}
	}
	if err := s.repo.DeleteCard(ctx, s.pool, cardID); err != nil {
		return err
	}
	s.notify(board.GroupID)
	return nil
}

// Move relocates a card to targetColumn at targetIndex, densely
// re-sequencing the affected columns.
func (s *Service) Move(ctx context.Context, cardID, targetColumnID uuid.UUID, targetIndex int, actor Actor) error {
	card, board, err := s.cardWithBoard(ctx, cardID)
	if err != nil {
		return err
	}
	if err := s.requireMember(ctx, board.GroupID, actor); err != nil {
		return err
	}
	target, err := s.repo.GetColumn(ctx, s.pool, targetColumnID)
	if err != nil {
		return err
	}
	if target.BoardID != board.ID {
		return ErrColumnMissing
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("boards: begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	if card.ColumnID == targetColumnID {
		ids, err := s.repo.ListCardIDsInColumn(ctx, tx, targetColumnID)
		if err != nil {
			return err
		}
		ids = remove(ids, cardID)
		ids = insertAt(ids, cardID, targetIndex)
		if err := reindex(ctx, tx, s.repo, targetColumnID, ids); err != nil {
			return err
		}
	} else {
		destIDs, err := s.repo.ListCardIDsInColumn(ctx, tx, targetColumnID)
		if err != nil {
			return err
		}
		destIDs = insertAt(destIDs, cardID, targetIndex)
		if err := reindex(ctx, tx, s.repo, targetColumnID, destIDs); err != nil {
			return err
		}

		srcIDs, err := s.repo.ListCardIDsInColumn(ctx, tx, card.ColumnID)
		if err != nil {
			return err
		}
		srcIDs = remove(srcIDs, cardID)
		if err := reindex(ctx, tx, s.repo, card.ColumnID, srcIDs); err != nil {
			return err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("boards: commit move: %w", err)
	}
	s.notify(board.GroupID)
	return nil
}

// --- helpers ---

func (s *Service) requireMember(ctx context.Context, groupID uuid.UUID, actor Actor) error {
	if err := s.access.EnsureActorMember(ctx, groupID, actor.UserID, actor.RoleLevel); err != nil {
		return ErrForbidden
	}
	return nil
}

func (s *Service) requireFounder(ctx context.Context, groupID uuid.UUID, actor Actor) error {
	if err := s.requireMember(ctx, groupID, actor); err != nil {
		return err
	}
	founder, err := s.access.IsFounder(ctx, groupID, actor.UserID)
	if err != nil {
		return err
	}
	if !founder {
		return ErrForbidden
	}
	return nil
}

func (s *Service) boardForFounder(ctx context.Context, boardID uuid.UUID, actor Actor) (*Board, error) {
	board, err := s.repo.GetBoardByID(ctx, s.pool, boardID)
	if err != nil {
		return nil, err
	}
	if err := s.requireFounder(ctx, board.GroupID, actor); err != nil {
		return nil, err
	}
	return board, nil
}

func (s *Service) cardWithBoard(ctx context.Context, cardID uuid.UUID) (*Card, *Board, error) {
	card, err := s.repo.GetCard(ctx, s.pool, cardID)
	if err != nil {
		return nil, nil, err
	}
	board, err := s.repo.GetBoardByID(ctx, s.pool, card.BoardID)
	if err != nil {
		return nil, nil, err
	}
	return card, board, nil
}

func (s *Service) validateAssignee(ctx context.Context, groupID uuid.UUID, assignee *uuid.UUID) error {
	if assignee == nil {
		return nil
	}
	ok, err := s.access.IsMember(ctx, groupID, *assignee)
	if err != nil {
		return err
	}
	if !ok {
		return ErrInvalidInput
	}
	return nil
}

func (s *Service) notify(groupID uuid.UUID) {
	if s.pub != nil {
		s.pub.PublishBoardChanged(groupID)
	}
}

// remove drops the first occurrence of id.
func remove(ids []uuid.UUID, id uuid.UUID) []uuid.UUID {
	out := ids[:0]
	for _, x := range ids {
		if x != id {
			out = append(out, x)
		}
	}
	return out
}

// insertAt inserts id at a clamped index.
func insertAt(ids []uuid.UUID, id uuid.UUID, index int) []uuid.UUID {
	if index < 0 {
		index = 0
	}
	if index > len(ids) {
		index = len(ids)
	}
	out := make([]uuid.UUID, 0, len(ids)+1)
	out = append(out, ids[:index]...)
	out = append(out, id)
	out = append(out, ids[index:]...)
	return out
}

// reindex writes dense positions (0..n-1) for the ordered cards, moving
// them into columnID.
func reindex(ctx context.Context, q db.DBTX, repo Repository, columnID uuid.UUID, ordered []uuid.UUID) error {
	for i, id := range ordered {
		if err := repo.SetCardPosition(ctx, q, id, columnID, i); err != nil {
			return err
		}
	}
	return nil
}
