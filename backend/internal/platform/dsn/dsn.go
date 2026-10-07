// Package dsn keeps connection-string passwords out of error messages.
//
// golang-migrate, net/url and go-redis quote a connection string they cannot
// parse in full, so one typo by an operator in DATABASE_URL or REDIS_URL put
// the managed database's password in the platform logs (audit A-39).
package dsn

import (
	"net/url"
	"regexp"
	"strings"
)

const mask = "xxxxx"

// keywordPassword matches password=… in a keyword/value DSN, quoted or not.
var keywordPassword = regexp.MustCompile(`password\s*=\s*('(?:[^'\\]|\\.)*'|\S+)`)

// Scrub returns err with every password of raw masked out of its message,
// or nil for a nil err. The result does not unwrap to err: the original text
// is exactly what must not reach a log.
func Scrub(err error, raw string) error {
	if err == nil {
		return nil
	}
	msg := err.Error()
	for _, s := range secrets(raw) {
		msg = strings.ReplaceAll(msg, s, mask)
	}
	return &scrubbed{msg: msg}
}

type scrubbed struct{ msg string }

func (e *scrubbed) Error() string { return e.msg }

// secrets lists the forms a password of raw can take in an error message —
// as written, and percent-decoded — longest first, so a password containing
// another one is masked whole.
func secrets(raw string) []string {
	var found []string
	add := func(s string) {
		s = strings.Trim(s, "'")
		if s == "" {
			return
		}
		found = append(found, s)
		if dec, err := url.PathUnescape(s); err == nil && dec != s {
			found = append(found, dec)
		}
		if dec, err := url.QueryUnescape(s); err == nil && dec != s {
			found = append(found, dec)
		}
	}

	if u, err := url.Parse(raw); err == nil && u.User != nil {
		if pw, ok := u.User.Password(); ok {
			add(pw)
			add(url.QueryEscape(pw))
		}
	} else if i := strings.Index(raw, "://"); i >= 0 {
		// Unparseable — the very case that leaks. The userinfo runs up to the
		// last '@' of the authority; the password is what follows its ':'.
		rest := raw[i+3:]
		if at := strings.LastIndex(rest, "@"); at >= 0 {
			if c := strings.Index(rest[:at], ":"); c >= 0 {
				add(rest[c+1 : at])
			}
		}
	}
	for _, m := range keywordPassword.FindAllStringSubmatch(raw, -1) {
		add(m[1])
	}

	// Longest first: masking "pw" before "pw2" would leave a "2" behind.
	for i := 1; i < len(found); i++ {
		for j := i; j > 0 && len(found[j]) > len(found[j-1]); j-- {
			found[j], found[j-1] = found[j-1], found[j]
		}
	}
	return found
}
