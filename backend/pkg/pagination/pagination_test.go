package pagination

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestEncodeDecodeRoundTrip(t *testing.T) {
	original := Cursor{CreatedAt: time.Now().UTC().Truncate(time.Microsecond), ID: uuid.New()}

	encoded := Encode(original)
	decoded, present, err := Decode(encoded)
	if err != nil {
		t.Fatalf("Decode() error: %v", err)
	}
	if !present {
		t.Fatal("expected cursor present")
	}
	if !decoded.CreatedAt.Equal(original.CreatedAt) || decoded.ID != original.ID {
		t.Fatalf("round trip mismatch: got %+v want %+v", decoded, original)
	}
}

func TestDecodeEmpty(t *testing.T) {
	c, present, err := Decode("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if present {
		t.Fatal("empty cursor should not be present")
	}
	if c != (Cursor{}) {
		t.Fatal("empty cursor should be zero value")
	}
}

func TestDecodeGarbage(t *testing.T) {
	for _, s := range []string{"!!!not-base64!!!", "YWJj"} { // 2nd is valid b64 but not JSON
		if _, _, err := Decode(s); err != ErrInvalidCursor {
			t.Errorf("Decode(%q) error = %v, want ErrInvalidCursor", s, err)
		}
	}
}

func TestNormalizeLimit(t *testing.T) {
	cases := []struct{ in, want int }{
		{0, DefaultLimit},
		{-5, DefaultLimit},
		{10, 10},
		{200, 200},
		{201, MaxLimit},
		{99999, MaxLimit},
	}
	for _, c := range cases {
		if got := NormalizeLimit(c.in); got != c.want {
			t.Errorf("NormalizeLimit(%d) = %d, want %d", c.in, got, c.want)
		}
	}
}
