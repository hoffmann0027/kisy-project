// Package pagination implements cursor-based pagination with stable
// ordering, per docs/spec/09-api-contracts.md ("Cursor-based pagination.
// Stable ordering. Default limit 50. Maximum 200.").
package pagination

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
)

const (
	DefaultLimit = 50
	MaxLimit     = 200
)

var ErrInvalidCursor = errors.New("pagination: invalid cursor")

// Cursor points at the last item of the previous page. Ordering is by
// (created_at DESC, id DESC); the id tie-breaker keeps ordering stable
// when several rows share a timestamp.
type Cursor struct {
	CreatedAt time.Time `json:"c"`
	ID        uuid.UUID `json:"i"`
}

// Encode serializes a cursor into an opaque URL-safe string.
func Encode(c Cursor) string {
	raw, _ := json.Marshal(c)
	return base64.RawURLEncoding.EncodeToString(raw)
}

// Decode parses an opaque cursor string. An empty string yields a zero
// cursor and no error, meaning "start from the beginning".
func Decode(s string) (Cursor, bool, error) {
	if s == "" {
		return Cursor{}, false, nil
	}
	raw, err := base64.RawURLEncoding.DecodeString(s)
	if err != nil {
		return Cursor{}, false, ErrInvalidCursor
	}
	var c Cursor
	if err := json.Unmarshal(raw, &c); err != nil {
		return Cursor{}, false, ErrInvalidCursor
	}
	return c, true, nil
}

// NormalizeLimit clamps a requested limit into [1, MaxLimit], defaulting
// to DefaultLimit when unset (<= 0).
func NormalizeLimit(requested int) int {
	switch {
	case requested <= 0:
		return DefaultLimit
	case requested > MaxLimit:
		return MaxLimit
	default:
		return requested
	}
}

// Page is a generic result page. NextCursor is empty when there are no
// more items.
type Page[T any] struct {
	Items      []T    `json:"items"`
	NextCursor string `json:"nextCursor,omitempty"`
	HasMore    bool   `json:"hasMore"`
}
