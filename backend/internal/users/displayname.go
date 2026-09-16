package users

import (
	"errors"
	"strings"
	"unicode"
)

// Display names: the one rule, used by registration, the profile, and the
// CEO bootstrap. The database holds the same rule (migration 46,
// kisy_display_name_valid) as a backstop; this is where it is applied, so a
// caller gets a clear error instead of a constraint violation.

var (
	// ErrDisplayNameLength: fewer than 2 or more than 40 letters and spaces.
	ErrDisplayNameLength = errors.New("users: display name must be 2-40 characters")
	// ErrDisplayNameCharacters: something other than letters and single spaces.
	ErrDisplayNameCharacters = errors.New("users: display name may contain only letters and single spaces")
	// ErrDisplayNameTaken: someone already has this name, ignoring case and spaces.
	ErrDisplayNameTaken = errors.New("users: display name already taken")
)

const (
	displayNameMin = 2
	displayNameMax = 40
)

// displayNameLetters is the alphabet — kept identical to the character class
// in migration 46: Latin (Basic, Latin-1 letters, Extended-A and -B) and the
// Cyrillic block without U+0482–U+0489, which are a numeric sign and
// combining marks rather than letters.
var displayNameLetters = &unicode.RangeTable{
	R16: []unicode.Range16{
		{Lo: 'A', Hi: 'Z', Stride: 1},
		{Lo: 'a', Hi: 'z', Stride: 1},
		{Lo: 0x00C0, Hi: 0x00D6, Stride: 1},
		{Lo: 0x00D8, Hi: 0x00F6, Stride: 1},
		{Lo: 0x00F8, Hi: 0x024F, Stride: 1},
		{Lo: 0x0400, Hi: 0x0481, Stride: 1},
		{Lo: 0x048A, Hi: 0x04FF, Stride: 1},
	},
	// Entries lying wholly within Latin-1 (Hi <= 0xFF), as unicode.Is expects.
	LatinOffset: 4,
}

// NormalizeDisplayName returns the name as it is stored: surrounding spaces
// removed and any run of whitespace between words reduced to one space. Then
// it checks the rule. Uniqueness is the database's to decide.
//
// Collapsing spaces is normalization, not rejection: "Анна  Смирнова" is the
// name "Анна Смирнова" typed with a stray space, and it is the same name for
// uniqueness either way.
func NormalizeDisplayName(raw string) (string, error) {
	name := strings.Join(strings.Fields(raw), " ")
	n := len([]rune(name))
	if n < displayNameMin || n > displayNameMax {
		return "", ErrDisplayNameLength
	}
	for _, r := range name {
		if r == ' ' {
			continue
		}
		if !unicode.Is(displayNameLetters, r) {
			return "", ErrDisplayNameCharacters
		}
	}
	return name, nil
}

// IsDisplayNameError reports whether err is one of the rule's own errors — the
// kind a handler turns into a 400/409 rather than a 500.
func IsDisplayNameError(err error) bool {
	return errors.Is(err, ErrDisplayNameLength) ||
		errors.Is(err, ErrDisplayNameCharacters) ||
		errors.Is(err, ErrDisplayNameTaken)
}
