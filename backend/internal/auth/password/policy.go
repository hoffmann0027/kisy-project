package password

import (
	"unicode"
	"unicode/utf8"
)

// The one password policy of this service. It used to be three: sign-up
// demanded a letter and a digit, the change-password screen only a length, and
// the CEO's reset a different length again — so an administratively reset
// password could be weaker than any user was allowed to choose (audit D-14).
const (
	// MinRunes/MaxRunes count characters, not bytes. Bytes were the older
	// rule, and they quietly made the limit language-dependent: a Cyrillic
	// password of 65 characters is 130 bytes and was refused as "too long".
	MinRunes = 12
	MaxRunes = 128
)

// RuleText is the wording shown to people and returned by the API, kept next
// to the rule so the two cannot drift. Mirrored in the frontend by
// shared/lib/password.ts.
const RuleText = "пароль: 12–128 символов, минимум одна буква и одна цифра"

// Valid reports whether p may be used as a password. Letters and digits are
// Unicode-wide on purpose: a Cyrillic password is a password.
func Valid(p string) bool {
	n := utf8.RuneCountInString(p)
	if n < MinRunes || n > MaxRunes {
		return false
	}
	var hasLetter, hasDigit bool
	for _, r := range p {
		switch {
		case unicode.IsLetter(r):
			hasLetter = true
		case unicode.IsDigit(r):
			hasDigit = true
		}
	}
	return hasLetter && hasDigit
}
