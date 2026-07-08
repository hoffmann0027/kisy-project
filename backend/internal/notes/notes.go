// Package notes owns the personal "Заметки" scratchpad: each user keeps
// private notes (free text and/or a single attached file) that only they can
// read or delete. Ownership is enforced in the service on every operation, so
// one user can never reach another's notes even by guessing an id.
package notes

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"kisy-backend/internal/platform/db"
)

const (
	// MaxTextLen caps a note's text; MaxFileBytes caps an attached file.
	MaxTextLen   = 10000
	MaxFileBytes = 10 << 20 // 10 MiB
)

var (
	ErrNotFound  = errors.New("notes: not found")
	ErrEmpty     = errors.New("notes: note has neither text nor file")
	ErrTooLong   = errors.New("notes: text too long")
	ErrTooLarge  = errors.New("notes: file exceeds size limit")
	ErrForbidden = errors.New("notes: not the owner")
)

// DTO is the API view of a note. It never carries the file bytes — those are
// fetched separately via the file endpoint.
type DTO struct {
	ID        uuid.UUID `json:"id"`
	Text      *string   `json:"text"`
	HasFile   bool      `json:"hasFile"`
	FileName  *string   `json:"fileName"`
	FileType  *string   `json:"fileType"`
	FileSize  int64     `json:"fileSize"`
	FileURL   *string   `json:"fileUrl"`
	CreatedAt time.Time `json:"createdAt"`
}

// Blob is a note file's bytes and metadata, for serving.
type Blob struct {
	FileName string
	FileType string
	Data     []byte
}

// Repository is the persistence port for notes.
type Repository interface {
	Create(ctx context.Context, q db.DBTX, userID uuid.UUID, text *string, fileName, fileType *string, fileSize int64, fileBytes []byte) (DTO, error)
	List(ctx context.Context, q db.DBTX, userID uuid.UUID) ([]DTO, error)
	// GetFile returns the file bytes of a note the user owns; ErrNotFound if
	// the note does not exist, is not the user's, or carries no file.
	GetFile(ctx context.Context, q db.DBTX, id, userID uuid.UUID) (Blob, error)
	// Delete removes a note the user owns; ErrNotFound otherwise.
	Delete(ctx context.Context, q db.DBTX, id, userID uuid.UUID) error
}

type PostgresRepository struct{}

func NewPostgresRepository() *PostgresRepository { return &PostgresRepository{} }

func scanRow(row pgx.Row) (DTO, error) {
	var d DTO
	var fileName, fileType *string
	if err := row.Scan(&d.ID, &d.Text, &fileName, &fileType, &d.FileSize, &d.HasFile, &d.CreatedAt); err != nil {
		return DTO{}, err
	}
	if d.HasFile {
		d.FileName = fileName
		d.FileType = fileType
		url := "/api/v1/notes/" + d.ID.String() + "/file"
		d.FileURL = &url
	}
	return d, nil
}

const selectCols = `id, text, file_name, file_type, file_size, (file_bytes IS NOT NULL) AS has_file, created_at`

func (r *PostgresRepository) Create(ctx context.Context, q db.DBTX, userID uuid.UUID, text *string, fileName, fileType *string, fileSize int64, fileBytes []byte) (DTO, error) {
	row := q.QueryRow(ctx, `
		INSERT INTO notes (user_id, text, file_name, file_type, file_size, file_bytes)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING `+selectCols,
		userID, text, fileName, fileType, fileSize, fileBytes)
	d, err := scanRow(row)
	if err != nil {
		return DTO{}, fmt.Errorf("notes: create: %w", err)
	}
	return d, nil
}

func (r *PostgresRepository) List(ctx context.Context, q db.DBTX, userID uuid.UUID) ([]DTO, error) {
	rows, err := q.Query(ctx, `
		SELECT `+selectCols+`
		FROM notes WHERE user_id = $1
		ORDER BY created_at DESC, id DESC`, userID)
	if err != nil {
		return nil, fmt.Errorf("notes: list: %w", err)
	}
	defer rows.Close()

	var out []DTO
	for rows.Next() {
		d, err := scanRow(rows)
		if err != nil {
			return nil, fmt.Errorf("notes: scan: %w", err)
		}
		out = append(out, d)
	}
	return out, rows.Err()
}

func (r *PostgresRepository) GetFile(ctx context.Context, q db.DBTX, id, userID uuid.UUID) (Blob, error) {
	var b Blob
	err := q.QueryRow(ctx, `
		SELECT file_name, file_type, file_bytes
		FROM notes WHERE id = $1 AND user_id = $2 AND file_bytes IS NOT NULL`,
		id, userID).Scan(&b.FileName, &b.FileType, &b.Data)
	if errors.Is(err, pgx.ErrNoRows) {
		return Blob{}, ErrNotFound
	}
	if err != nil {
		return Blob{}, fmt.Errorf("notes: get file: %w", err)
	}
	return b, nil
}

func (r *PostgresRepository) Delete(ctx context.Context, q db.DBTX, id, userID uuid.UUID) error {
	tag, err := q.Exec(ctx, `DELETE FROM notes WHERE id = $1 AND user_id = $2`, id, userID)
	if err != nil {
		return fmt.Errorf("notes: delete: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// Service holds notes business logic. Every method is scoped to the acting
// user's own notes.
type Service struct {
	pool *pgxpool.Pool
	repo Repository
}

func NewService(pool *pgxpool.Pool, repo Repository) *Service {
	return &Service{pool: pool, repo: repo}
}

// CreateText stores a text-only note.
func (s *Service) CreateText(ctx context.Context, userID uuid.UUID, text string) (DTO, error) {
	text = strings.TrimSpace(text)
	if text == "" {
		return DTO{}, ErrEmpty
	}
	if len([]rune(text)) > MaxTextLen {
		return DTO{}, ErrTooLong
	}
	return s.repo.Create(ctx, s.pool, userID, &text, nil, nil, 0, nil)
}

// CreateFile stores a note carrying a file, with optional caption text. The
// content type is sniffed from the bytes, not trusted from the client.
func (s *Service) CreateFile(ctx context.Context, userID uuid.UUID, fileName, caption string, raw []byte) (DTO, error) {
	if len(raw) == 0 {
		return DTO{}, ErrEmpty
	}
	if len(raw) > MaxFileBytes {
		return DTO{}, ErrTooLarge
	}
	caption = strings.TrimSpace(caption)
	if len([]rune(caption)) > MaxTextLen {
		return DTO{}, ErrTooLong
	}
	var text *string
	if caption != "" {
		text = &caption
	}
	name := strings.TrimSpace(fileName)
	if name == "" {
		name = "file"
	}
	mime := http.DetectContentType(raw)
	return s.repo.Create(ctx, s.pool, userID, text, &name, &mime, int64(len(raw)), raw)
}

// List returns the user's notes, newest first.
func (s *Service) List(ctx context.Context, userID uuid.UUID) ([]DTO, error) {
	items, err := s.repo.List(ctx, s.pool, userID)
	if err != nil {
		return nil, err
	}
	if items == nil {
		items = []DTO{}
	}
	return items, nil
}

// FetchFile returns a note's file bytes if the actor owns the note.
func (s *Service) FetchFile(ctx context.Context, id, userID uuid.UUID) (Blob, error) {
	return s.repo.GetFile(ctx, s.pool, id, userID)
}

// Delete removes one of the user's own notes.
func (s *Service) Delete(ctx context.Context, id, userID uuid.UUID) error {
	return s.repo.Delete(ctx, s.pool, id, userID)
}
