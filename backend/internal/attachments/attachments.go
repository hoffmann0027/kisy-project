// Package attachments stores message file/image attachments in the database
// (so they survive redeploys on ephemeral disks) and serves them only to users
// who can access the carrying message.
package attachments

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"kisy-backend/internal/platform/db"
)

const MaxBytes = 10 << 20 // 10 MiB per file

var (
	ErrNotFound = errors.New("attachments: not found")
	ErrTooLarge = errors.New("attachments: file exceeds size limit")
	ErrEmpty    = errors.New("attachments: empty file")
)

// DTO is the public description of an attachment (never its bytes).
type DTO struct {
	ID        uuid.UUID `json:"id"`
	FileName  string    `json:"fileName"`
	MimeType  string    `json:"mimeType"`
	SizeBytes int64     `json:"sizeBytes"`
	IsImage   bool      `json:"isImage"`
	URL       string    `json:"url"`
}

func toDTO(id uuid.UUID, name, mime string, size int64) DTO {
	return DTO{
		ID:        id,
		FileName:  name,
		MimeType:  mime,
		SizeBytes: size,
		IsImage:   strings.HasPrefix(mime, "image/"),
		URL:       "/api/v1/attachments/" + id.String(),
	}
}

// Blob is a stored file's bytes and metadata (for serving).
type Blob struct {
	MimeType   string
	FileName   string
	Data       []byte
	MessageID  *uuid.UUID
	UploadedBy *uuid.UUID
}

// Repository is the persistence port.
type Repository interface {
	Create(ctx context.Context, q db.DBTX, name, mime string, size int64, data []byte, uploadedBy uuid.UUID) (uuid.UUID, error)
	Link(ctx context.Context, q db.DBTX, ids []uuid.UUID, messageID, uploader uuid.UUID) error
	Get(ctx context.Context, q db.DBTX, id uuid.UUID) (Blob, error)
	ForMessages(ctx context.Context, q db.DBTX, messageIDs []uuid.UUID) (map[uuid.UUID][]DTO, error)
}

type PostgresRepository struct{}

func NewPostgresRepository() *PostgresRepository { return &PostgresRepository{} }

func (r *PostgresRepository) Create(ctx context.Context, q db.DBTX, name, mime string, size int64, data []byte, uploadedBy uuid.UUID) (uuid.UUID, error) {
	var id uuid.UUID
	err := q.QueryRow(ctx, `
		INSERT INTO attachments (message_id, file_name, mime_type, size_bytes, storage_path, scan_status, data, uploaded_by)
		VALUES (NULL, $1, $2, $3, '', 'clean', $4, $5)
		RETURNING id`,
		name, mime, size, data, uploadedBy).Scan(&id)
	if err != nil {
		return uuid.Nil, fmt.Errorf("attachments: create: %w", err)
	}
	return id, nil
}

func (r *PostgresRepository) Link(ctx context.Context, q db.DBTX, ids []uuid.UUID, messageID, uploader uuid.UUID) error {
	if len(ids) == 0 {
		return nil
	}
	_, err := q.Exec(ctx, `
		UPDATE attachments SET message_id = $2
		WHERE id = ANY($1) AND message_id IS NULL AND uploaded_by = $3`,
		ids, messageID, uploader)
	if err != nil {
		return fmt.Errorf("attachments: link: %w", err)
	}
	return nil
}

func (r *PostgresRepository) Get(ctx context.Context, q db.DBTX, id uuid.UUID) (Blob, error) {
	var b Blob
	err := q.QueryRow(ctx, `SELECT mime_type, file_name, data, message_id, uploaded_by FROM attachments WHERE id = $1`, id).
		Scan(&b.MimeType, &b.FileName, &b.Data, &b.MessageID, &b.UploadedBy)
	if errors.Is(err, pgx.ErrNoRows) {
		return Blob{}, ErrNotFound
	}
	if err != nil {
		return Blob{}, fmt.Errorf("attachments: get: %w", err)
	}
	return b, nil
}

func (r *PostgresRepository) ForMessages(ctx context.Context, q db.DBTX, messageIDs []uuid.UUID) (map[uuid.UUID][]DTO, error) {
	out := make(map[uuid.UUID][]DTO)
	if len(messageIDs) == 0 {
		return out, nil
	}
	rows, err := q.Query(ctx, `
		SELECT message_id, id, file_name, mime_type, size_bytes
		FROM attachments WHERE message_id = ANY($1)
		ORDER BY created_at`, messageIDs)
	if err != nil {
		return nil, fmt.Errorf("attachments: for messages: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var mid, id uuid.UUID
		var name, mime string
		var size int64
		if err := rows.Scan(&mid, &id, &name, &mime, &size); err != nil {
			return nil, fmt.Errorf("attachments: scan: %w", err)
		}
		out[mid] = append(out[mid], toDTO(id, name, mime, size))
	}
	return out, rows.Err()
}

// MessageAccess reports whether an actor may see a given message's chat.
// Injected to avoid an attachments→messages import cycle.
type MessageAccess func(ctx context.Context, messageID, actorID uuid.UUID, actorLevel int) bool

type Service struct {
	pool   *pgxpool.Pool
	repo   Repository
	access MessageAccess
}

func NewService(pool *pgxpool.Pool, repo Repository) *Service {
	return &Service{pool: pool, repo: repo}
}

// SetMessageAccess wires the access check used when serving linked files.
func (s *Service) SetMessageAccess(a MessageAccess) { s.access = a }

// Upload validates and stores a file, returning its metadata. The content type
// is sniffed from the bytes, not trusted from the client.
func (s *Service) Upload(ctx context.Context, fileName string, raw []byte, uploader uuid.UUID) (DTO, error) {
	if len(raw) == 0 {
		return DTO{}, ErrEmpty
	}
	if len(raw) > MaxBytes {
		return DTO{}, ErrTooLarge
	}
	mime := http.DetectContentType(raw)
	name := strings.TrimSpace(fileName)
	if name == "" {
		name = "file"
	}
	id, err := s.repo.Create(ctx, s.pool, name, mime, int64(len(raw)), raw, uploader)
	if err != nil {
		return DTO{}, err
	}
	return toDTO(id, name, mime, int64(len(raw))), nil
}

// Link attaches uploaded files to a message (best-effort ownership-checked).
func (s *Service) Link(ctx context.Context, q db.DBTX, ids []uuid.UUID, messageID, uploader uuid.UUID) error {
	return s.repo.Link(ctx, q, ids, messageID, uploader)
}

// ForMessages returns attachment DTOs grouped by message id, for enrichment.
func (s *Service) ForMessages(ctx context.Context, messageIDs []uuid.UUID) (map[uuid.UUID][]DTO, error) {
	return s.repo.ForMessages(ctx, s.pool, messageIDs)
}

// Fetch returns a file's bytes if the actor may access it: a linked file needs
// access to its message's chat; an unlinked file only its uploader.
func (s *Service) Fetch(ctx context.Context, id, actorID uuid.UUID, actorLevel int) (Blob, error) {
	b, err := s.repo.Get(ctx, s.pool, id)
	if err != nil {
		return Blob{}, err
	}
	if b.MessageID != nil {
		if s.access == nil || !s.access(ctx, *b.MessageID, actorID, actorLevel) {
			return Blob{}, ErrNotFound
		}
		return b, nil
	}
	if b.UploadedBy == nil || *b.UploadedBy != actorID {
		return Blob{}, ErrNotFound
	}
	return b, nil
}
