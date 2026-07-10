package attachments

import (
	"errors"
	"testing"
)

func i32(v int32) *int32 { return &v }

func TestInspectBlocksExecutables(t *testing.T) {
	cases := map[string][]byte{
		"windows PE": append([]byte("MZ"), make([]byte, 64)...),
		"ELF":        append([]byte{0x7f, 'E', 'L', 'F'}, make([]byte, 64)...),
		"Mach-O":     append([]byte{0xfe, 0xed, 0xfa, 0xcf}, make([]byte, 64)...),
	}
	for name, raw := range cases {
		if _, err := inspect(raw); !errors.Is(err, ErrBlockedType) {
			t.Errorf("%s: want ErrBlockedType, got %v", name, err)
		}
	}

	png := append([]byte{0x89, 'P', 'N', 'G', 0x0d, 0x0a, 0x1a, 0x0a}, make([]byte, 64)...)
	mime, err := inspect(png)
	if err != nil || mime != "image/png" {
		t.Fatalf("png: mime=%q err=%v", mime, err)
	}
}

func TestMetaNormalize(t *testing.T) {
	// Kind inference from sniffed MIME.
	m, err := Meta{}.normalize("image/png")
	if err != nil || m.Kind != KindImage {
		t.Fatalf("image inference: %+v %v", m, err)
	}
	m, err = Meta{}.normalize("application/pdf")
	if err != nil || m.Kind != KindFile {
		t.Fatalf("file inference: %+v %v", m, err)
	}

	// Voice with duration and waveform is valid.
	if _, err := (Meta{Kind: KindVoice, DurationMs: i32(3000), Waveform: []byte{1, 2, 3}}).normalize("audio/webm"); err != nil {
		t.Fatalf("voice meta: %v", err)
	}

	// Cross-check violations.
	if _, err := (Meta{Kind: KindFile, DurationMs: i32(1)}).normalize("application/pdf"); !errors.Is(err, ErrBadMeta) {
		t.Fatalf("duration on file: want ErrBadMeta, got %v", err)
	}
	if _, err := (Meta{Kind: KindVoice, Width: i32(10)}).normalize("audio/webm"); !errors.Is(err, ErrBadMeta) {
		t.Fatalf("width on voice: want ErrBadMeta, got %v", err)
	}
	if _, err := (Meta{Kind: "weird"}).normalize("application/pdf"); !errors.Is(err, ErrBadMeta) {
		t.Fatalf("unknown kind: want ErrBadMeta, got %v", err)
	}
	if _, err := (Meta{Kind: KindVoice, Waveform: make([]byte, MaxWaveformBytes+1)}).normalize("audio/webm"); !errors.Is(err, ErrBadMeta) {
		t.Fatalf("oversized waveform: want ErrBadMeta, got %v", err)
	}
	if _, err := (Meta{Kind: KindImage, Width: i32(-2)}).normalize("image/png"); !errors.Is(err, ErrBadMeta) {
		t.Fatalf("negative width: want ErrBadMeta, got %v", err)
	}
}

func TestLimitsByClearance(t *testing.T) {
	l := Limits{MaxBytesLeadership: 200, MaxBytesStaff: 50, LeadershipMaxLevel: 3}
	if l.MaxBytesFor(1) != 200 || l.MaxBytesFor(3) != 200 {
		t.Fatalf("leadership levels must get the larger ceiling")
	}
	if l.MaxBytesFor(4) != 50 || l.MaxBytesFor(10) != 50 {
		t.Fatalf("staff levels must get the smaller ceiling")
	}
}

func TestChunkCount(t *testing.T) {
	s := UploadSession{DeclaredBytes: 25, ChunkBytes: 10}
	if s.chunkCount() != 3 {
		t.Fatalf("25/10 = %d chunks, want 3", s.chunkCount())
	}
	s = UploadSession{DeclaredBytes: 20, ChunkBytes: 10}
	if s.chunkCount() != 2 {
		t.Fatalf("20/10 = %d chunks, want 2", s.chunkCount())
	}
	s = UploadSession{DeclaredBytes: 1, ChunkBytes: 10}
	if s.chunkCount() != 1 {
		t.Fatalf("1/10 = %d chunks, want 1", s.chunkCount())
	}
}
