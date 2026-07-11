//go:build integration

package attachments_test

import (
	"bytes"
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"kisy-backend/internal/attachments"
	"kisy-backend/internal/platform/testdb"
)

const (
	testChunk   = 10 << 10  // 10 KiB
	staffLimit  = 100 << 10 // 100 KiB
	leaderLimit = 500 << 10
	staffLevel  = 8
	leaderLevel = 2
)

func setup(t *testing.T) (*attachments.Service, uuid.UUID, context.Context) {
	pool := testdb.New(t)
	svc := attachments.NewService(pool, attachments.NewPostgresRepository(), attachments.Limits{
		MaxBytesLeadership: leaderLimit,
		MaxBytesStaff:      staffLimit,
		LeadershipMaxLevel: 3,
		ChunkBytes:         testChunk,
		SessionTTL:         time.Hour,
	})
	uploader := testdb.SeedUser(t, pool, "uploader", staffLevel)
	return svc, uploader, context.Background()
}

// pattern returns deterministic non-trivial bytes of the given size.
func pattern(size int) []byte {
	out := make([]byte, size)
	for i := range out {
		out[i] = byte(i*7 + i/255)
	}
	return out
}

func TestChunkedUploadHappyPath(t *testing.T) {
	svc, uploader, ctx := setup(t)
	raw := pattern(25 << 10) // 25 KiB → chunks of 10/10/5

	session, err := svc.InitUpload(ctx, uploader, staffLevel, "report.bin", int64(len(raw)), attachments.Meta{})
	if err != nil {
		t.Fatalf("init: %v", err)
	}
	if session.ChunkBytes != testChunk {
		t.Fatalf("chunk size = %d, want %d", session.ChunkBytes, testChunk)
	}

	// Chunks arrive out of order; one is re-sent (idempotent).
	chunks := [][]byte{raw[:testChunk], raw[testChunk : 2*testChunk], raw[2*testChunk:]}
	for _, idx := range []int{2, 0, 1, 0} {
		if err := svc.PutChunk(ctx, uploader, session.ID, idx, chunks[idx]); err != nil {
			t.Fatalf("put chunk %d: %v", idx, err)
		}
	}

	status, err := svc.UploadStatus(ctx, uploader, session.ID)
	if err != nil {
		t.Fatalf("status: %v", err)
	}
	if len(status.ReceivedChunks) != 3 {
		t.Fatalf("received = %v, want 3 chunks", status.ReceivedChunks)
	}

	dto, err := svc.CompleteUpload(ctx, uploader, session.ID)
	if err != nil {
		t.Fatalf("complete: %v", err)
	}
	if dto.SizeBytes != int64(len(raw)) || dto.Kind != attachments.KindFile {
		t.Fatalf("dto mismatch: %+v", dto)
	}

	// The assembled bytes round-trip through Fetch.
	blob, err := svc.Fetch(ctx, dto.ID, uploader, staffLevel)
	if err != nil {
		t.Fatalf("fetch: %v", err)
	}
	if !bytes.Equal(blob.Data, raw) {
		t.Fatalf("assembled bytes differ from original")
	}

	// The session is gone after complete.
	if _, err := svc.UploadStatus(ctx, uploader, session.ID); !errors.Is(err, attachments.ErrNotFound) {
		t.Fatalf("session should be deleted, got %v", err)
	}
}

func TestChunkedUploadValidation(t *testing.T) {
	svc, uploader, ctx := setup(t)

	// Declared size over the staff limit is rejected at init.
	if _, err := svc.InitUpload(ctx, uploader, staffLevel, "big.bin", staffLimit+1, attachments.Meta{}); !errors.Is(err, attachments.ErrTooLarge) {
		t.Fatalf("over staff limit: want ErrTooLarge, got %v", err)
	}
	// The same size passes for leadership.
	if _, err := svc.InitUpload(ctx, uploader, leaderLevel, "big.bin", staffLimit+1, attachments.Meta{}); err != nil {
		t.Fatalf("leadership init: %v", err)
	}

	raw := pattern(15 << 10) // 2 chunks: 10 KiB + 5 KiB
	session, err := svc.InitUpload(ctx, uploader, staffLevel, "doc.bin", int64(len(raw)), attachments.Meta{})
	if err != nil {
		t.Fatalf("init: %v", err)
	}

	// A non-final chunk must fill the chunk size exactly.
	if err := svc.PutChunk(ctx, uploader, session.ID, 0, raw[:100]); !errors.Is(err, attachments.ErrBadChunk) {
		t.Fatalf("short non-final chunk: want ErrBadChunk, got %v", err)
	}
	// Out-of-range index.
	if err := svc.PutChunk(ctx, uploader, session.ID, 5, raw[:testChunk]); !errors.Is(err, attachments.ErrBadChunk) {
		t.Fatalf("index out of range: want ErrBadChunk, got %v", err)
	}

	// Complete with a missing chunk fails.
	if err := svc.PutChunk(ctx, uploader, session.ID, 0, raw[:testChunk]); err != nil {
		t.Fatalf("put chunk 0: %v", err)
	}
	if _, err := svc.CompleteUpload(ctx, uploader, session.ID); !errors.Is(err, attachments.ErrIncomplete) {
		t.Fatalf("incomplete: want ErrIncomplete, got %v", err)
	}

	// A stranger can neither see nor touch the session.
	stranger := uuid.New()
	if _, err := svc.UploadStatus(ctx, stranger, session.ID); !errors.Is(err, attachments.ErrNotFound) {
		t.Fatalf("stranger status: want ErrNotFound, got %v", err)
	}
	if err := svc.PutChunk(ctx, stranger, session.ID, 1, raw[testChunk:]); !errors.Is(err, attachments.ErrNotFound) {
		t.Fatalf("stranger chunk: want ErrNotFound, got %v", err)
	}
}

func TestChunkedUploadBlocksExecutables(t *testing.T) {
	svc, uploader, ctx := setup(t)

	// A Windows PE split across two chunks must be rejected at complete —
	// inspection runs on the assembled bytes, before the file is servable.
	raw := append([]byte("MZ"), pattern(15<<10-2)...)
	session, err := svc.InitUpload(ctx, uploader, staffLevel, "totally-a-doc.pdf", int64(len(raw)), attachments.Meta{})
	if err != nil {
		t.Fatalf("init: %v", err)
	}
	if err := svc.PutChunk(ctx, uploader, session.ID, 0, raw[:testChunk]); err != nil {
		t.Fatalf("chunk 0: %v", err)
	}
	if err := svc.PutChunk(ctx, uploader, session.ID, 1, raw[testChunk:]); err != nil {
		t.Fatalf("chunk 1: %v", err)
	}
	if _, err := svc.CompleteUpload(ctx, uploader, session.ID); !errors.Is(err, attachments.ErrBlockedType) {
		t.Fatalf("executable: want ErrBlockedType, got %v", err)
	}

	// Single-shot path applies the same inspection.
	if _, err := svc.Upload(ctx, "evil.exe", raw, uploader, staffLevel, attachments.Meta{}); !errors.Is(err, attachments.ErrBlockedType) {
		t.Fatalf("single-shot executable: want ErrBlockedType, got %v", err)
	}
}

func TestSingleShotLimitsAndMeta(t *testing.T) {
	svc, uploader, ctx := setup(t)

	// Over the staff limit single-shot is rejected; leadership passes.
	big := pattern(staffLimit + 1)
	if _, err := svc.Upload(ctx, "big.bin", big, uploader, staffLevel, attachments.Meta{}); !errors.Is(err, attachments.ErrTooLarge) {
		t.Fatalf("staff single-shot: want ErrTooLarge, got %v", err)
	}
	if _, err := svc.Upload(ctx, "big.bin", big, uploader, leaderLevel, attachments.Meta{}); err != nil {
		t.Fatalf("leadership single-shot: %v", err)
	}

	// Voice meta round-trips through storage and message enrichment. The
	// bytes carry the EBML/WebM magic so the sniffer sees a real container.
	webm := append([]byte{0x1a, 0x45, 0xdf, 0xa3}, pattern(2044)...)
	dto, err := svc.Upload(ctx, "note.webm", webm, uploader, staffLevel, attachments.Meta{
		Kind: attachments.KindVoice, DurationMs: i32(4200), Waveform: []byte{9, 8, 7},
	})
	if err != nil {
		t.Fatalf("voice upload: %v", err)
	}
	if dto.Kind != attachments.KindVoice || dto.DurationMs == nil || *dto.DurationMs != 4200 || len(dto.Waveform) != 3 {
		t.Fatalf("voice dto mismatch: %+v", dto)
	}
}

func i32(v int32) *int32 { return &v }
