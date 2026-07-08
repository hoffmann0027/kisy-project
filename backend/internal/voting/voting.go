// Package voting owns the "Голосование" board: the CEO creates polls (a
// description plus up to ten options); every member casts one changeable vote
// while a poll is open. Votes are public — each option carries the avatars of
// the users who chose it. Closing a poll (CEO only) freezes it and writes the
// final tally to the audit log. Authorization is enforced in the service.
package voting

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
	"kisy-backend/internal/platform/db"
)

const (
	MaxQuestionLen = 500
	MaxOptionLen   = 200
	MinOptions     = 2
	MaxOptions     = 10
)

var (
	ErrForbidden  = errors.New("voting: not permitted")
	ErrValidation = errors.New("voting: invalid input")
	ErrNotFound   = errors.New("voting: not found")
	ErrClosed     = errors.New("voting: poll is closed")
)

// Voter is the public identity shown under an option they picked.
type Voter struct {
	ID          uuid.UUID `json:"id"`
	DisplayName string    `json:"displayName"`
	AvatarURL   *string   `json:"avatarUrl"`
}

// OptionDTO is one choice with its tally and voters.
type OptionDTO struct {
	ID     uuid.UUID `json:"id"`
	Body   string    `json:"body"`
	Votes  int       `json:"votes"`
	Voters []Voter   `json:"voters"`
}

// PollDTO is a poll with its options and the caller's own choice.
type PollDTO struct {
	ID         uuid.UUID   `json:"id"`
	Question   string      `json:"question"`
	Status     string      `json:"status"`
	Options    []OptionDTO `json:"options"`
	TotalVotes int         `json:"totalVotes"`
	MyOptionID *uuid.UUID  `json:"myOptionId"`
	CreatedAt  time.Time   `json:"createdAt"`
	ClosedAt   *time.Time  `json:"closedAt"`
}

// Actor identifies the acting user for authorization.
type Actor struct {
	UserID    uuid.UUID
	RoleLevel int
}

func (a Actor) isCEO() bool { return a.RoleLevel == 1 }

// Repository is the persistence port.
type Repository interface {
	ListPolls(ctx context.Context, q db.DBTX) ([]pollRow, error)
	ListOptions(ctx context.Context, q db.DBTX) ([]optionRow, error)
	ListVotes(ctx context.Context, q db.DBTX) ([]voteRow, error)
	CreatePoll(ctx context.Context, q db.DBTX, question string, options []string, createdBy uuid.UUID) (uuid.UUID, error)
	OptionPoll(ctx context.Context, q db.DBTX, optionID uuid.UUID) (uuid.UUID, string, error)
	PollStatus(ctx context.Context, q db.DBTX, pollID uuid.UUID) (string, error)
	Vote(ctx context.Context, q db.DBTX, pollID, optionID, userID uuid.UUID) error
	Close(ctx context.Context, q db.DBTX, pollID uuid.UUID) error
	Delete(ctx context.Context, q db.DBTX, pollID uuid.UUID) error
	// Tally returns option bodies with their vote counts, for the audit log.
	Tally(ctx context.Context, q db.DBTX, pollID uuid.UUID) (string, []OptionTally, error)
}

type pollRow struct {
	ID        uuid.UUID
	Question  string
	Status    string
	CreatedAt time.Time
	ClosedAt  *time.Time
}

type optionRow struct {
	ID     uuid.UUID
	PollID uuid.UUID
	Body   string
}

type voteRow struct {
	PollID      uuid.UUID
	OptionID    uuid.UUID
	UserID      uuid.UUID
	DisplayName string
	AvatarURL   *string
}

// OptionTally is one line of a poll's final result.
type OptionTally struct {
	Body  string
	Votes int
}

type PostgresRepository struct{}

func NewPostgresRepository() *PostgresRepository { return &PostgresRepository{} }

func (r *PostgresRepository) ListPolls(ctx context.Context, q db.DBTX) ([]pollRow, error) {
	rows, err := q.Query(ctx, `SELECT id, question, status, created_at, closed_at FROM polls ORDER BY created_at DESC, id DESC`)
	if err != nil {
		return nil, fmt.Errorf("voting: list polls: %w", err)
	}
	defer rows.Close()
	var out []pollRow
	for rows.Next() {
		var p pollRow
		if err := rows.Scan(&p.ID, &p.Question, &p.Status, &p.CreatedAt, &p.ClosedAt); err != nil {
			return nil, fmt.Errorf("voting: scan poll: %w", err)
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

func (r *PostgresRepository) ListOptions(ctx context.Context, q db.DBTX) ([]optionRow, error) {
	rows, err := q.Query(ctx, `SELECT id, poll_id, body FROM poll_options ORDER BY poll_id, position`)
	if err != nil {
		return nil, fmt.Errorf("voting: list options: %w", err)
	}
	defer rows.Close()
	var out []optionRow
	for rows.Next() {
		var o optionRow
		if err := rows.Scan(&o.ID, &o.PollID, &o.Body); err != nil {
			return nil, fmt.Errorf("voting: scan option: %w", err)
		}
		out = append(out, o)
	}
	return out, rows.Err()
}

func (r *PostgresRepository) ListVotes(ctx context.Context, q db.DBTX) ([]voteRow, error) {
	rows, err := q.Query(ctx, `
		SELECT v.poll_id, v.option_id, v.user_id, u.display_name, u.avatar_url
		FROM poll_votes v JOIN users u ON u.id = v.user_id
		ORDER BY v.created_at`)
	if err != nil {
		return nil, fmt.Errorf("voting: list votes: %w", err)
	}
	defer rows.Close()
	var out []voteRow
	for rows.Next() {
		var v voteRow
		if err := rows.Scan(&v.PollID, &v.OptionID, &v.UserID, &v.DisplayName, &v.AvatarURL); err != nil {
			return nil, fmt.Errorf("voting: scan vote: %w", err)
		}
		out = append(out, v)
	}
	return out, rows.Err()
}

func (r *PostgresRepository) CreatePoll(ctx context.Context, q db.DBTX, question string, options []string, createdBy uuid.UUID) (uuid.UUID, error) {
	var id uuid.UUID
	if err := q.QueryRow(ctx, `INSERT INTO polls (question, created_by) VALUES ($1, $2) RETURNING id`, question, createdBy).Scan(&id); err != nil {
		return uuid.Nil, fmt.Errorf("voting: create poll: %w", err)
	}
	for i, body := range options {
		if _, err := q.Exec(ctx, `INSERT INTO poll_options (poll_id, position, body) VALUES ($1, $2, $3)`, id, i, body); err != nil {
			return uuid.Nil, fmt.Errorf("voting: create option: %w", err)
		}
	}
	return id, nil
}

func (r *PostgresRepository) OptionPoll(ctx context.Context, q db.DBTX, optionID uuid.UUID) (uuid.UUID, string, error) {
	var pollID uuid.UUID
	var body string
	err := q.QueryRow(ctx, `SELECT poll_id, body FROM poll_options WHERE id = $1`, optionID).Scan(&pollID, &body)
	if errors.Is(err, pgx.ErrNoRows) {
		return uuid.Nil, "", ErrNotFound
	}
	if err != nil {
		return uuid.Nil, "", fmt.Errorf("voting: option poll: %w", err)
	}
	return pollID, body, nil
}

func (r *PostgresRepository) PollStatus(ctx context.Context, q db.DBTX, pollID uuid.UUID) (string, error) {
	var status string
	err := q.QueryRow(ctx, `SELECT status FROM polls WHERE id = $1`, pollID).Scan(&status)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrNotFound
	}
	if err != nil {
		return "", fmt.Errorf("voting: poll status: %w", err)
	}
	return status, nil
}

func (r *PostgresRepository) Vote(ctx context.Context, q db.DBTX, pollID, optionID, userID uuid.UUID) error {
	_, err := q.Exec(ctx, `
		INSERT INTO poll_votes (poll_id, option_id, user_id) VALUES ($1, $2, $3)
		ON CONFLICT (poll_id, user_id) DO UPDATE SET option_id = EXCLUDED.option_id, created_at = now()`,
		pollID, optionID, userID)
	if err != nil {
		return fmt.Errorf("voting: vote: %w", err)
	}
	return nil
}

func (r *PostgresRepository) Close(ctx context.Context, q db.DBTX, pollID uuid.UUID) error {
	tag, err := q.Exec(ctx, `UPDATE polls SET status = 'closed', closed_at = now() WHERE id = $1 AND status = 'open'`, pollID)
	if err != nil {
		return fmt.Errorf("voting: close: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *PostgresRepository) Delete(ctx context.Context, q db.DBTX, pollID uuid.UUID) error {
	tag, err := q.Exec(ctx, `DELETE FROM polls WHERE id = $1`, pollID)
	if err != nil {
		return fmt.Errorf("voting: delete: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *PostgresRepository) Tally(ctx context.Context, q db.DBTX, pollID uuid.UUID) (string, []OptionTally, error) {
	var question string
	err := q.QueryRow(ctx, `SELECT question FROM polls WHERE id = $1`, pollID).Scan(&question)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", nil, ErrNotFound
	}
	if err != nil {
		return "", nil, fmt.Errorf("voting: tally question: %w", err)
	}
	rows, err := q.Query(ctx, `
		SELECT o.body, count(v.user_id)
		FROM poll_options o
		LEFT JOIN poll_votes v ON v.option_id = o.id
		WHERE o.poll_id = $1
		GROUP BY o.id, o.body, o.position
		ORDER BY o.position`, pollID)
	if err != nil {
		return "", nil, fmt.Errorf("voting: tally: %w", err)
	}
	defer rows.Close()
	var out []OptionTally
	for rows.Next() {
		var t OptionTally
		if err := rows.Scan(&t.Body, &t.Votes); err != nil {
			return "", nil, fmt.Errorf("voting: scan tally: %w", err)
		}
		out = append(out, t)
	}
	return question, out, rows.Err()
}

// ChangePublisher notifies every connected client that the poll board changed.
// Injected to avoid a voting→ws cycle; may be nil.
type ChangePublisher func()

// Service holds the voting business logic.
type Service struct {
	pool    *pgxpool.Pool
	repo    Repository
	audit   audit.Recorder
	changed ChangePublisher
}

func NewService(pool *pgxpool.Pool, repo Repository, rec audit.Recorder) *Service {
	return &Service{pool: pool, repo: repo, audit: rec}
}

func (s *Service) SetChangePublisher(p ChangePublisher) { s.changed = p }

func (s *Service) notify() {
	if s.changed != nil {
		s.changed()
	}
}

// List returns every poll with options, tallies, voters and the caller's vote.
func (s *Service) List(ctx context.Context, actor Actor) ([]PollDTO, error) {
	polls, err := s.repo.ListPolls(ctx, s.pool)
	if err != nil {
		return nil, err
	}
	options, err := s.repo.ListOptions(ctx, s.pool)
	if err != nil {
		return nil, err
	}
	votes, err := s.repo.ListVotes(ctx, s.pool)
	if err != nil {
		return nil, err
	}

	// Index options per poll (preserving order) and build option DTOs.
	optionsByPoll := make(map[uuid.UUID][]*OptionDTO)
	optionByID := make(map[uuid.UUID]*OptionDTO)
	for _, o := range options {
		dto := &OptionDTO{ID: o.ID, Body: o.Body, Voters: []Voter{}}
		optionsByPoll[o.PollID] = append(optionsByPoll[o.PollID], dto)
		optionByID[o.ID] = dto
	}

	// myVote[pollID] = the option the actor chose in that poll.
	myVote := make(map[uuid.UUID]uuid.UUID)
	totalByPoll := make(map[uuid.UUID]int)
	for _, v := range votes {
		if dto, ok := optionByID[v.OptionID]; ok {
			dto.Votes++
			dto.Voters = append(dto.Voters, Voter{ID: v.UserID, DisplayName: v.DisplayName, AvatarURL: v.AvatarURL})
		}
		totalByPoll[v.PollID]++
		if v.UserID == actor.UserID {
			myVote[v.PollID] = v.OptionID
		}
	}

	out := make([]PollDTO, 0, len(polls))
	for _, p := range polls {
		dto := PollDTO{
			ID: p.ID, Question: p.Question, Status: p.Status,
			CreatedAt: p.CreatedAt, ClosedAt: p.ClosedAt,
			TotalVotes: totalByPoll[p.ID], Options: []OptionDTO{},
		}
		for _, o := range optionsByPoll[p.ID] {
			dto.Options = append(dto.Options, *o)
		}
		if opt, ok := myVote[p.ID]; ok {
			mine := opt
			dto.MyOptionID = &mine
		}
		out = append(out, dto)
	}
	return out, nil
}

// Create adds a poll. CEO only. Requires 2..10 non-empty options.
func (s *Service) Create(ctx context.Context, question string, options []string, actor Actor) (uuid.UUID, error) {
	if !actor.isCEO() {
		return uuid.Nil, ErrForbidden
	}
	question = strings.TrimSpace(question)
	if n := len([]rune(question)); n < 1 || n > MaxQuestionLen {
		return uuid.Nil, ErrValidation
	}
	cleaned := make([]string, 0, len(options))
	for _, o := range options {
		o = strings.TrimSpace(o)
		if o == "" {
			continue
		}
		if len([]rune(o)) > MaxOptionLen {
			return uuid.Nil, ErrValidation
		}
		cleaned = append(cleaned, o)
	}
	if len(cleaned) < MinOptions || len(cleaned) > MaxOptions {
		return uuid.Nil, ErrValidation
	}
	id, err := s.repo.CreatePoll(ctx, s.pool, question, cleaned, actor.UserID)
	if err == nil {
		s.notify()
	}
	return id, err
}

// Vote records (or changes) the actor's choice. Any user may vote while the
// poll is open. The option must belong to the poll it names.
func (s *Service) Vote(ctx context.Context, optionID uuid.UUID, actor Actor) error {
	pollID, _, err := s.repo.OptionPoll(ctx, s.pool, optionID)
	if err != nil {
		return err
	}
	status, err := s.repo.PollStatus(ctx, s.pool, pollID)
	if err != nil {
		return err
	}
	if status != "open" {
		return ErrClosed
	}
	err = s.repo.Vote(ctx, s.pool, pollID, optionID, actor.UserID)
	if err == nil {
		s.notify()
	}
	return err
}

// Close freezes a poll (CEO only) and records the final tally in the audit log.
func (s *Service) Close(ctx context.Context, pollID uuid.UUID, actor Actor) error {
	if !actor.isCEO() {
		return ErrForbidden
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	if err := s.repo.Close(ctx, tx, pollID); err != nil {
		return err
	}
	question, tally, err := s.repo.Tally(ctx, tx, pollID)
	if err != nil {
		return err
	}

	results := make([]map[string]any, len(tally))
	winner, best := "", -1
	total := 0
	for i, t := range tally {
		results[i] = map[string]any{"option": t.Body, "votes": t.Votes}
		total += t.Votes
		if t.Votes > best {
			best, winner = t.Votes, t.Body
		}
	}
	actorID := actor.UserID
	pid := pollID
	if err := s.audit.Record(ctx, tx, audit.Event{
		ActorID:    &actorID,
		Action:     ActionPollResult,
		TargetType: "poll",
		TargetID:   &pid,
		Metadata: map[string]any{
			"question":    question,
			"results":     results,
			"totalVotes":  total,
			"winner":      winner,
			"winnerVotes": best,
		},
	}); err != nil {
		return err
	}

	if err := tx.Commit(ctx); err != nil {
		return err
	}
	s.notify()
	return nil
}

// Delete removes a poll and its votes. CEO only.
func (s *Service) Delete(ctx context.Context, pollID uuid.UUID, actor Actor) error {
	if !actor.isCEO() {
		return ErrForbidden
	}
	err := s.repo.Delete(ctx, s.pool, pollID)
	if err == nil {
		s.notify()
	}
	return err
}

// ActionPollResult is the audit action recorded when a poll is closed.
const ActionPollResult = "poll.result"
