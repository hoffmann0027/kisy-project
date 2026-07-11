// Package attachments stores message file/image attachments in the database
// (so they survive redeploys on ephemeral disks) and serves them only to users
// who can access the carrying message.
//
// Two upload paths exist (stage A): single-shot POST /attachments for small
// files, and a resumable chunked flow (init → chunk → complete) for large
// ones. Both apply the same size limit (clearance-differentiated, from
// config) and the same content inspection before a file becomes servable.
package attachments

import (
	"bytes"
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

var (
	ErrNotFound    = errors.New("attachments: not found")
	ErrTooLarge    = errors.New("attachments: file exceeds size limit")
	ErrEmpty       = errors.New("attachments: empty file")
	ErrBadMeta     = errors.New("attachments: invalid media metadata")
	ErrBlockedType = errors.New("attachments: file type is not allowed")
	ErrBadChunk    = errors.New("attachments: invalid chunk")
	ErrIncomplete  = errors.New("attachments: upload is missing chunks")
)

// Attachment kinds (attachments.kind).
const (
	KindFile  = "file"
	KindImage = "image"
	KindVoice = "voice"
	KindVideo = "video"
)

// MaxWaveformBytes mirrors the DB CHECK on the waveform envelope.
const MaxWaveformBytes = 1024

// MaxVoiceDurationMs bounds a voice note (10 minutes).
const MaxVoiceDurationMs = 10 * 60 * 1000

// voiceMimes are the sniffed content types accepted for voice notes —
// containers MediaRecorder actually produces. Audio-only WebM shares the
// container magic with video, so Go's sniffer reports video/webm; same for
// MP4 (Safari).
var voiceMimes = map[string]bool{
	"video/webm":      true,
	"audio/webm":      true,
	"application/ogg": true,
	"audio/ogg":       true,
	"video/mp4":       true,
	"audio/mp4":       true,
	"audio/mpeg":      true,
	"audio/wave":      true,
	"audio/wav":       true,
}

// Limits is the upload policy injected from config (never hardcoded):
// leadership levels (1..LeadershipMaxLevel) get the larger ceiling.
type Limits struct {
	MaxBytesLeadership int64
	MaxBytesStaff      int64
	LeadershipMaxLevel int
	ChunkBytes         int
	SessionTTL         time.Duration
}

// MaxBytesFor returns the upload ceiling for a clearance level.
func (l Limits) MaxBytesFor(roleLevel int) int64 {
	if roleLevel <= l.LeadershipMaxLevel {
		return l.MaxBytesLeadership
	}
	return l.MaxBytesStaff
}

// Meta carries client-declared media properties. The kind is validated and,
// when empty, inferred from the sniffed MIME; waveform/duration/dimensions
// power voice bubbles and the media viewer.
type Meta struct {
	Kind       string `json:"kind"`
	DurationMs *int32 `json:"durationMs"`
	Waveform   []byte `json:"waveform"` // base64 in JSON
	Width      *int32 `json:"width"`
	Height     *int32 `json:"height"`
}

// validateShape rejects malformed meta regardless of content: unknown kind,
// oversized waveform, non-positive numbers. Cheap enough for init-time.
func (m Meta) validateShape() error {
	switch m.Kind {
	case "", KindFile, KindImage, KindVoice, KindVideo:
	default:
		return ErrBadMeta
	}
	if len(m.Waveform) > MaxWaveformBytes {
		return ErrBadMeta
	}
	if m.DurationMs != nil && *m.DurationMs < 0 {
		return ErrBadMeta
	}
	if (m.Width != nil && *m.Width <= 0) || (m.Height != nil && *m.Height <= 0) {
		return ErrBadMeta
	}
	return nil
}

// normalize validates the meta against the sniffed MIME and fills the kind.
func (m Meta) normalize(sniffedMime string) (Meta, error) {
	if err := m.validateShape(); err != nil {
		return Meta{}, err
	}
	if m.Kind == "" {
		if strings.HasPrefix(sniffedMime, "image/") {
			m.Kind = KindImage
		} else {
			m.Kind = KindFile
		}
	}
	// Duration/waveform only make sense for playable media; dimensions only
	// for visual media. Reject mismatches instead of storing junk.
	if (m.DurationMs != nil || len(m.Waveform) > 0) && m.Kind != KindVoice && m.Kind != KindVideo {
		return Meta{}, ErrBadMeta
	}
	if (m.Width != nil || m.Height != nil) && m.Kind != KindImage && m.Kind != KindVideo {
		return Meta{}, ErrBadMeta
	}
	// A voice note must actually be audio (stage B): the sniffed container
	// must be one MediaRecorder produces, and the duration is mandatory and
	// bounded — clients rely on both for the player UI.
	if m.Kind == KindVoice {
		if !voiceMimes[sniffedMime] {
			return Meta{}, ErrBadMeta
		}
		if m.DurationMs == nil || *m.DurationMs == 0 || *m.DurationMs > MaxVoiceDurationMs {
			return Meta{}, ErrBadMeta
		}
	}
	return m, nil
}

// executableMagics identify native executables, which have no legitimate
// place in a corporate chat and are the highest-risk payload. Inspection
// runs on the assembled bytes BEFORE a file becomes servable — on both the
// single-shot and the chunked path (docs/spec/06-security.md).
var executableMagics = [][]byte{
	{'M', 'Z'},                          // Windows PE
	{0x7f, 'E', 'L', 'F'},               // ELF
	{0xfe, 0xed, 0xfa, 0xce},            // Mach-O 32
	{0xfe, 0xed, 0xfa, 0xcf},            // Mach-O 64
	{0xcf, 0xfa, 0xed, 0xfe},            // Mach-O 64 (LE)
	{0xca, 0xfe, 0xba, 0xbe},            // Mach-O fat / Java class
	{'!', '<', 'a', 'r', 'c', 'h', '>'}, // static library
}

// inspect sniffs the real content type and rejects executable payloads.
func inspect(raw []byte) (mime string, err error) {
	for _, magic := range executableMagics {
		if bytes.HasPrefix(raw, magic) {
			return "", ErrBlockedType
		}
	}
	return http.DetectContentType(raw), nil
}

// DTO is the public description of an attachment (never its bytes).
type DTO struct {
	ID        uuid.UUID `json:"id"`
	FileName  string    `json:"fileName"`
	MimeType  string    `json:"mimeType"`
	SizeBytes int64     `json:"sizeBytes"`
	IsImage   bool      `json:"isImage"`
	URL       string    `json:"url"`

	Kind       string `json:"kind"`
	DurationMs *int32 `json:"durationMs,omitempty"`
	Waveform   []byte `json:"waveform,omitempty"` // base64 in JSON
	Width      *int32 `json:"width,omitempty"`
	Height     *int32 `json:"height,omitempty"`
}

func toDTO(id uuid.UUID, name, mime string, size int64, meta Meta) DTO {
	return DTO{
		ID:         id,
		FileName:   name,
		MimeType:   mime,
		SizeBytes:  size,
		IsImage:    strings.HasPrefix(mime, "image/"),
		URL:        "/api/v1/attachments/" + id.String(),
		Kind:       meta.Kind,
		DurationMs: meta.DurationMs,
		Waveform:   meta.Waveform,
		Width:      meta.Width,
		Height:     meta.Height,
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
	Create(ctx context.Context, q db.DBTX, name, mime string, size int64, data []byte, uploadedBy uuid.UUID, meta Meta) (uuid.UUID, error)
	Link(ctx context.Context, q db.DBTX, ids []uuid.UUID, messageID, uploader uuid.UUID) error
	Get(ctx context.Context, q db.DBTX, id uuid.UUID) (Blob, error)
	ForMessages(ctx context.Context, q db.DBTX, messageIDs []uuid.UUID) (map[uuid.UUID][]DTO, error)
	// CopyToMessage duplicates every attachment of sourceMessageID onto
	// newMessageID (new rows, same bytes/metadata), owned by uploader, and
	// returns the new attachment DTOs. Used by message forwarding (stage D).
	CopyToMessage(ctx context.Context, q db.DBTX, sourceMessageID, newMessageID, uploader uuid.UUID) ([]DTO, error)

	CreateSession(ctx context.Context, q db.DBTX, s *UploadSession) error
	GetSession(ctx context.Context, q db.DBTX, id uuid.UUID) (*UploadSession, error)
	PutChunk(ctx context.Context, q db.DBTX, sessionID uuid.UUID, idx int, data []byte) error
	// ChunkIndexes returns the stored chunk indexes (ascending) and total bytes.
	ChunkIndexes(ctx context.Context, q db.DBTX, sessionID uuid.UUID) ([]int, int64, error)
	AssembleChunks(ctx context.Context, q db.DBTX, sessionID uuid.UUID) ([]byte, error)
	DeleteSession(ctx context.Context, q db.DBTX, id uuid.UUID) error
	DeleteExpiredSessions(ctx context.Context, q db.DBTX, now time.Time) (int64, error)
}

type PostgresRepository struct{}

func NewPostgresRepository() *PostgresRepository { return &PostgresRepository{} }

func (r *PostgresRepository) Create(ctx context.Context, q db.DBTX, name, mime string, size int64, data []byte, uploadedBy uuid.UUID, meta Meta) (uuid.UUID, error) {
	var id uuid.UUID
	err := q.QueryRow(ctx, `
		INSERT INTO attachments (message_id, file_name, mime_type, size_bytes, storage_path, scan_status, data, uploaded_by,
		                         kind, duration_ms, waveform, width, height)
		VALUES (NULL, $1, $2, $3, '', 'clean', $4, $5, $6, $7, $8, $9, $10)
		RETURNING id`,
		name, mime, size, data, uploadedBy,
		meta.Kind, meta.DurationMs, meta.Waveform, meta.Width, meta.Height).Scan(&id)
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
		SELECT message_id, id, file_name, mime_type, size_bytes, kind, duration_ms, waveform, width, height
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
		var meta Meta
		if err := rows.Scan(&mid, &id, &name, &mime, &size, &meta.Kind, &meta.DurationMs, &meta.Waveform, &meta.Width, &meta.Height); err != nil {
			return nil, fmt.Errorf("attachments: scan: %w", err)
		}
		out[mid] = append(out[mid], toDTO(id, name, mime, size, meta))
	}
	return out, rows.Err()
}

func (r *PostgresRepository) CopyToMessage(ctx context.Context, q db.DBTX, sourceMessageID, newMessageID, uploader uuid.UUID) ([]DTO, error) {
	rows, err := q.Query(ctx, `
		INSERT INTO attachments (message_id, file_name, mime_type, size_bytes, storage_path, scan_status, data, uploaded_by,
		                         kind, duration_ms, waveform, width, height)
		SELECT $2, file_name, mime_type, size_bytes, '', 'clean', data, $3,
		       kind, duration_ms, waveform, width, height
		FROM attachments WHERE message_id = $1
		ORDER BY created_at
		RETURNING id, file_name, mime_type, size_bytes, kind, duration_ms, waveform, width, height`,
		sourceMessageID, newMessageID, uploader)
	if err != nil {
		return nil, fmt.Errorf("attachments: copy to message: %w", err)
	}
	defer rows.Close()

	var out []DTO
	for rows.Next() {
		var id uuid.UUID
		var name, mime string
		var size int64
		var meta Meta
		if err := rows.Scan(&id, &name, &mime, &size, &meta.Kind, &meta.DurationMs, &meta.Waveform, &meta.Width, &meta.Height); err != nil {
			return nil, fmt.Errorf("attachments: scan copied: %w", err)
		}
		out = append(out, toDTO(id, name, mime, size, meta))
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
	limits Limits
}

func NewService(pool *pgxpool.Pool, repo Repository, limits Limits) *Service {
	return &Service{pool: pool, repo: repo, limits: limits}
}

// SetMessageAccess wires the access check used when serving linked files.
func (s *Service) SetMessageAccess(a MessageAccess) { s.access = a }

// Limits returns the upload policy for a clearance level (for clients).
func (s *Service) Limits(roleLevel int) (maxBytes int64, chunkBytes int) {
	return s.limits.MaxBytesFor(roleLevel), s.limits.ChunkBytes
}

// Upload validates and stores a file in one shot. The content type is sniffed
// from the bytes, not trusted from the client.
func (s *Service) Upload(ctx context.Context, fileName string, raw []byte, uploader uuid.UUID, roleLevel int, meta Meta) (DTO, error) {
	if len(raw) == 0 {
		return DTO{}, ErrEmpty
	}
	if int64(len(raw)) > s.limits.MaxBytesFor(roleLevel) {
		return DTO{}, ErrTooLarge
	}
	return s.store(ctx, fileName, raw, uploader, meta)
}

// store inspects assembled bytes and persists them — shared by both paths.
func (s *Service) store(ctx context.Context, fileName string, raw []byte, uploader uuid.UUID, meta Meta) (DTO, error) {
	mime, err := inspect(raw)
	if err != nil {
		return DTO{}, err
	}
	normalized, err := meta.normalize(mime)
	if err != nil {
		return DTO{}, err
	}
	name := strings.TrimSpace(fileName)
	if name == "" {
		name = "file"
	}
	id, err := s.repo.Create(ctx, s.pool, name, mime, int64(len(raw)), raw, uploader, normalized)
	if err != nil {
		return DTO{}, err
	}
	return toDTO(id, name, mime, int64(len(raw)), normalized), nil
}

// Link attaches uploaded files to a message (best-effort ownership-checked).
func (s *Service) Link(ctx context.Context, q db.DBTX, ids []uuid.UUID, messageID, uploader uuid.UUID) error {
	return s.repo.Link(ctx, q, ids, messageID, uploader)
}

// OwnedUnlinked filters ids down to attachments that still exist, belong to
// the uploader and are not linked to a message yet — i.e. exactly the set a
// Link call would bind. The scheduled-send worker (stage I) uses it to drop
// attachments that disappeared between scheduling and send time.
func (s *Service) OwnedUnlinked(ctx context.Context, ids []uuid.UUID, uploader uuid.UUID) ([]uuid.UUID, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	rows, err := s.pool.Query(ctx, `
		SELECT id FROM attachments
		WHERE id = ANY($1) AND message_id IS NULL AND uploaded_by = $2`,
		ids, uploader)
	if err != nil {
		return nil, fmt.Errorf("attachments: owned unlinked: %w", err)
	}
	defer rows.Close()
	present := make(map[uuid.UUID]struct{}, len(ids))
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("attachments: scan owned: %w", err)
		}
		present[id] = struct{}{}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	// Preserve the caller's order.
	out := make([]uuid.UUID, 0, len(present))
	for _, id := range ids {
		if _, ok := present[id]; ok {
			out = append(out, id)
		}
	}
	return out, nil
}

// CopyToMessage duplicates a source message's attachments onto a new message
// (message forwarding). Access to the source is the caller's responsibility —
// the forwarding service checks it before calling.
func (s *Service) CopyToMessage(ctx context.Context, sourceMessageID, newMessageID, uploader uuid.UUID) ([]DTO, error) {
	return s.repo.CopyToMessage(ctx, s.pool, sourceMessageID, newMessageID, uploader)
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
