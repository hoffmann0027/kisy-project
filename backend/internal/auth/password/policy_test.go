package password

import (
	"strings"
	"testing"
)

// TestValidIsOnePolicyForEveryScreen pins audit D-14. Three places checked a
// password and all three disagreed: sign-up wanted a letter and a digit, the
// change-password screen only a length, the CEO's reset yet another length —
// and the length was counted in bytes, which quietly refused a Cyrillic
// password of 65 characters as "too long" while the form allowed it.
func TestValidIsOnePolicyForEveryScreen(t *testing.T) {
	cases := []struct {
		name  string
		input string
		want  bool
	}{
		{"latin with a digit", "korrekt-parol1", true},
		{"cyrillic with a digit", "парольнадежный1", true},
		{"exactly the minimum", "abcdefghijk1", true},
		{"one character short", "abcdefghij1", false},
		{"no digit", "парольбезцифры", false},
		{"no letter", "123456789012", false},
		{"empty", "", false},
		// 128 Cyrillic characters: 256 bytes. The byte rule refused it.
		{"long cyrillic within the character limit", strings.Repeat("я", 127) + "1", true},
		{"one character over the limit", strings.Repeat("a", 128) + "1", false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := Valid(tc.input); got != tc.want {
				t.Fatalf("Valid(%q) = %v, want %v", tc.input, got, tc.want)
			}
		})
	}
}
