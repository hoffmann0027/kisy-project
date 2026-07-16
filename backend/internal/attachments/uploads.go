package attachments

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"kisy-backend/internal/platform/db"
	"kisy-backend/internal/platform/metrics"
)

// UploadSession is a chunked upload in progress: metadata is fixed at init,
// chunks arrive idempotently by index, complete assembles + inspects + stores.
type UploadSession struct {
	ID            uuid.UUID
	Uploader      uuid.UUID
	FileName      string
	DeclaredBytes int64
	ChunkBytes    int
	Meta          Meta
	CreatedAt     time.Time
	ExpiresAt     time.Time
}

// chunkCount is how many chunks a session needs to be complete.
func (s *UploadSession) chunkCount() int {
	return int((s.DeclaredBytes + int64(s.ChunkBytes) - 1) / int64(s.ChunkBytes))
}

// SessionDTO is what the client needs to drive the chunk loop.
type SessionDTO struct {
	ID            uuid.UUID `json:"id"`
	ChunkBytes    int       `json:"chunkBytes"`
	DeclaredBytes int64     `json:"declaredBytes"`
	ExpiresAt     time.Time `json:"expiresAt"`
	// ReceivedChunks lets an interrupted client resume: indexes already stored.
	ReceivedChunks []int `json:"receivedChunks"`
}

// InitUpload opens a chunked upload session after validating the declared
// size against the actor's clearance-based limit — cheap rejection before any
// bytes travel.
func (s *Service) InitUpload(ctx context.Context, uploader uuid.UUID, roleLevel int, fileName string, declaredBytes int64, meta Meta) (SessionDTO, error) {
	if declaredBytes <= 0 {
		return SessionDTO{}, ErrEmpty
	}
	if declaredBytes > s.limits.MaxBytesFor(roleLevel) {
		return SessionDTO{}, ErrTooLarge
	}
	// Meta shape is validated now; kind/media cross-checks happen at
	// complete, once the real MIME is known. Fail fast on garbage.
	if err := meta.validateShape(); err != nil {
		return SessionDTO{}, err
	}

	session := &UploadSession{
		Uploader:      uploader,
		FileName:      fileName,
		DeclaredBytes: declaredBytes,
		ChunkBytes:    s.limits.ChunkBytes,
		Meta:          meta,
		ExpiresAt:     time.Now().Add(s.limits.SessionTTL),
	}
	if err := s.repo.CreateSession(ctx, s.pool, session); err != nil {
		return SessionDTO{}, err
	}
	return SessionDTO{
		ID:             session.ID,
		ChunkBytes:     session.ChunkBytes,
		DeclaredBytes:  session.DeclaredBytes,
		ExpiresAt:      session.ExpiresAt,
		ReceivedChunks: []int{},
	}, nil
}

// ownedSession loads a live session belonging to the actor. Foreign or
// expired sessions read as not-found — no existence leaks.
func (s *Service) ownedSession(ctx context.Context, q db.DBTX, uploader, id uuid.UUID) (*UploadSession, error) {
	session, err := s.repo.GetSession(ctx, q, id)
	if err != nil {
		return nil, err
	}
	if session.Uploader != uploader || time.Now().After(session.ExpiresAt) {
		return nil, ErrNotFound
	}
	return session, nil
}

// UploadStatus reports stored chunk indexes so an interrupted client resumes
// instead of re-sending everything.
func (s *Service) UploadStatus(ctx context.Context, uploader, id uuid.UUID) (SessionDTO, error) {
	session, err := s.ownedSession(ctx, s.pool, uploader, id)
	if err != nil {
		return SessionDTO{}, err
	}
	received, _, err := s.repo.ChunkIndexes(ctx, s.pool, id)
	if err != nil {
		return SessionDTO{}, err
	}
	return SessionDTO{
		ID:             session.ID,
		ChunkBytes:     session.ChunkBytes,
		DeclaredBytes:  session.DeclaredBytes,
		ExpiresAt:      session.ExpiresAt,
		ReceivedChunks: received,
	}, nil
}

// PutChunk stores one chunk, idempotently by index: re-sending a chunk after
// a network hiccup simply overwrites the same bytes.
func (s *Service) PutChunk(ctx context.Context, uploader, id uuid.UUID, idx int, data []byte) error {
	session, err := s.ownedSession(ctx, s.pool, uploader, id)
	if err != nil {
		return err
	}
	count := session.chunkCount()
	if idx < 0 || idx >= count || len(data) == 0 || len(data) > session.ChunkBytes {
		return ErrBadChunk
	}
	// Every chunk except the last must fill the chunk size exactly, and the
	// last must land the total on declared_bytes — sizes are fixed upfront so
	// complete can verify integrity without trusting the client.
	if idx < count-1 && len(data) != session.ChunkBytes {
		return ErrBadChunk
	}
	if idx == count-1 && int64(len(data)) != session.DeclaredBytes-int64(count-1)*int64(session.ChunkBytes) {
		return ErrBadChunk
	}
	return s.repo.PutChunk(ctx, s.pool, id, idx, data)
}

// CompleteUpload assembles the chunks, runs the same inspection as the
// single-shot path and promotes the bytes into a real attachment. The
// session and its chunks are removed in the same transaction.
func (s *Service) CompleteUpload(ctx context.Context, uploader, id uuid.UUID) (DTO, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return DTO{}, fmt.Errorf("attachments: begin complete: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	session, err := s.ownedSession(ctx, tx, uploader, id)
	if err != nil {
		return DTO{}, err
	}
	received, total, err := s.repo.ChunkIndexes(ctx, tx, id)
	if err != nil {
		return DTO{}, err
	}
	count := session.chunkCount()
	if len(received) != count || total != session.DeclaredBytes {
		return DTO{}, ErrIncomplete
	}
	for i, idx := range received {
		if idx != i {
			return DTO{}, ErrIncomplete
		}
	}

	raw, err := s.repo.AssembleChunks(ctx, tx, id)
	if err != nil {
		return DTO{}, err
	}
	mime, err := inspect(raw)
	if err != nil {
		return DTO{}, err
	}
	meta, err := session.Meta.normalize(mime)
	if err != nil {
		return DTO{}, err
	}
	name := session.FileName
	if name == "" {
		name = "file"
	}
	attID, err := s.repo.Create(ctx, tx, name, mime, int64(len(raw)), raw, uploader, meta)
	if err != nil {
		return DTO{}, err
	}
	if err := s.repo.DeleteSession(ctx, tx, id); err != nil {
		return DTO{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return DTO{}, fmt.Errorf("attachments: commit complete: %w", err)
	}
	return toDTO(attID, name, mime, int64(len(raw)), meta), nil
}

// StartSessionCleanup reaps expired upload sessions (and their chunks, via
// cascade) on a fixed interval until ctx is cancelled.
func (s *Service) StartSessionCleanup(ctx context.Context, interval time.Duration, log *slog.Logger) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				metrics.WorkerRun("attachments_cleanup")
				n, err := s.repo.DeleteExpiredSessions(ctx, s.pool, time.Now())
				if err != nil && ctx.Err() == nil {
					metrics.WorkerError("attachments_cleanup")
					log.Warn("attachments: session cleanup failed", "error", err)
				} else if n > 0 {
					metrics.WorkerItems("attachments_cleanup", int(n))
					log.Info("attachments: reaped expired upload sessions", "count", n)
				}
			}
		}
	}()
}

// --- repository: upload sessions ---

func (r *PostgresRepository) CreateSession(ctx context.Context, q db.DBTX, s *UploadSession) error {
	err := q.QueryRow(ctx, `
		INSERT INTO attachment_upload_sessions
			(uploader, file_name, declared_bytes, chunk_bytes, kind, duration_ms, waveform, width, height, expires_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING id, created_at`,
		s.Uploader, s.FileName, s.DeclaredBytes, s.ChunkBytes,
		nonEmptyKind(s.Meta.Kind), s.Meta.DurationMs, s.Meta.Waveform, s.Meta.Width, s.Meta.Height,
		s.ExpiresAt,
	).Scan(&s.ID, &s.CreatedAt)
	if err != nil {
		return fmt.Errorf("attachments: create session: %w", err)
	}
	return nil
}

// nonEmptyKind maps "infer at complete" (empty) onto the DB default.
func nonEmptyKind(kind string) string {
	if kind == "" {
		return KindFile
	}
	return kind
}

func (r *PostgresRepository) GetSession(ctx context.Context, q db.DBTX, id uuid.UUID) (*UploadSession, error) {
	var s UploadSession
	err := q.QueryRow(ctx, `
		SELECT id, uploader, file_name, declared_bytes, chunk_bytes, kind, duration_ms, waveform, width, height, created_at, expires_at
		FROM attachment_upload_sessions WHERE id = $1`, id).
		Scan(&s.ID, &s.Uploader, &s.FileName, &s.DeclaredBytes, &s.ChunkBytes,
			&s.Meta.Kind, &s.Meta.DurationMs, &s.Meta.Waveform, &s.Meta.Width, &s.Meta.Height,
			&s.CreatedAt, &s.ExpiresAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("attachments: get session: %w", err)
	}
	return &s, nil
}

func (r *PostgresRepository) PutChunk(ctx context.Context, q db.DBTX, sessionID uuid.UUID, idx int, data []byte) error {
	_, err := q.Exec(ctx, `
		INSERT INTO attachment_upload_chunks (session_id, idx, data)
		VALUES ($1, $2, $3)
		ON CONFLICT (session_id, idx) DO UPDATE SET data = EXCLUDED.data`,
		sessionID, idx, data)
	if err != nil {
		return fmt.Errorf("attachments: put chunk: %w", err)
	}
	return nil
}

func (r *PostgresRepository) ChunkIndexes(ctx context.Context, q db.DBTX, sessionID uuid.UUID) ([]int, int64, error) {
	rows, err := q.Query(ctx, `
		SELECT idx, octet_length(data) FROM attachment_upload_chunks
		WHERE session_id = $1 ORDER BY idx`, sessionID)
	if err != nil {
		return nil, 0, fmt.Errorf("attachments: chunk indexes: %w", err)
	}
	defer rows.Close()

	indexes := []int{}
	var total int64
	for rows.Next() {
		var idx, size int
		if err := rows.Scan(&idx, &size); err != nil {
			return nil, 0, fmt.Errorf("attachments: scan chunk index: %w", err)
		}
		indexes = append(indexes, idx)
		total += int64(size)
	}
	return indexes, total, rows.Err()
}

func (r *PostgresRepository) AssembleChunks(ctx context.Context, q db.DBTX, sessionID uuid.UUID) ([]byte, error) {
	rows, err := q.Query(ctx, `
		SELECT data FROM attachment_upload_chunks
		WHERE session_id = $1 ORDER BY idx`, sessionID)
	if err != nil {
		return nil, fmt.Errorf("attachments: assemble: %w", err)
	}
	defer rows.Close()

	var out []byte
	for rows.Next() {
		var chunk []byte
		if err := rows.Scan(&chunk); err != nil {
			return nil, fmt.Errorf("attachments: scan chunk: %w", err)
		}
		out = append(out, chunk...)
	}
	return out, rows.Err()
}

func (r *PostgresRepository) DeleteSession(ctx context.Context, q db.DBTX, id uuid.UUID) error {
	_, err := q.Exec(ctx, `DELETE FROM attachment_upload_sessions WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("attachments: delete session: %w", err)
	}
	return nil
}

func (r *PostgresRepository) DeleteExpiredSessions(ctx context.Context, q db.DBTX, now time.Time) (int64, error) {
	tag, err := q.Exec(ctx, `DELETE FROM attachment_upload_sessions WHERE expires_at < $1`, now)
	if err != nil {
		return 0, fmt.Errorf("attachments: delete expired sessions: %w", err)
	}
	return tag.RowsAffected(), nil
}
