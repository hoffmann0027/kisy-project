// Package i18n holds the words the server itself says to people: refusals a
// screen shows as they are, and the pushes it writes. Everything else — the
// interface — is translated in the app; this is only what the app cannot
// know in advance.
//
// A request carries its language in the X-Kisy-Lang header (the one on the
// app's screen); a push goes out in the recipient's saved language. Russian is
// the source and the default: an app too old to say its language was a
// Russian one.
package i18n

import (
	"context"
	"fmt"
	"net/http"
	"strings"
)

// Lang is a language the server can answer in.
type Lang string

// Default is the language of a request that names none it knows.
const Default Lang = "ru"

// Supported lists every language, in the app's order. Must match
// frontend/src/shared/i18n/langs.ts.
var Supported = []Lang{"ru", "en", "de", "es", "fr", "nl", "pl", "cs", "uk", "tr"}

// Catalog is one language's messages, by key. A countable phrase has one
// entry per plural category: "key#one", "key#few", "key#many", "key#other".
type Catalog map[string]string

// catalogs holds every language with a catalog. A translation registers
// itself from its own file (func init), so translators never touch this one.
var catalogs = map[Lang]Catalog{
	"ru": ru,
	"en": en,
}

// Parse returns the supported language a tag names ("en-US" → en), or
// Default.
func Parse(tag string) Lang {
	primary := strings.ToLower(strings.TrimSpace(tag))
	if i := strings.IndexAny(primary, "-_;"); i >= 0 {
		primary = primary[:i]
	}
	for _, l := range Supported {
		if string(l) == primary {
			return l
		}
	}
	return Default
}

// IsSupported reports whether code is exactly one of the languages.
func IsSupported(code string) bool {
	for _, l := range Supported {
		if string(l) == code {
			return true
		}
	}
	return false
}

// Header carries the language on the app's screen. Not Accept-Language: a
// browser or WebView fills that with the device's languages on its own, and
// an app too old to know about languages would get answers in a language its
// Russian screen does not use.
const Header = "X-Kisy-Lang"

// FromRequest is the language the app on the other end shows, or Default.
func FromRequest(r *http.Request) Lang {
	return Parse(r.Header.Get(Header))
}

func lookup(l Lang, key string) (string, bool) {
	if s, ok := catalogs[l][key]; ok {
		return s, true
	}
	s, ok := catalogs[Default][key]
	return s, ok
}

// T is the message for key in l, formatted with args (fmt verbs). A key no
// catalog has comes back as itself, so a slip is visible rather than blank.
func T(l Lang, key string, args ...any) string {
	s, ok := lookup(l, key)
	if !ok {
		return key
	}
	if len(args) == 0 {
		return s
	}
	return fmt.Sprintf(s, localize(l, args)...)
}

// Localized is an argument that is itself worded per language: a nested
// message, a date in the reader's format.
type Localized interface {
	In(Lang) string
}

func localize(l Lang, args []any) []any {
	out := make([]any, len(args))
	for i, a := range args {
		if loc, ok := a.(Localized); ok {
			out[i] = loc.In(l)
		} else {
			out[i] = a
		}
	}
	return out
}

// N is the countable message for key and n in l: the plural form the
// language's rules pick, formatted with n first and then args.
func N(l Lang, key string, n int, args ...any) string {
	form := key + "#" + pluralCategory(l, n)
	s, ok := lookup(l, form)
	if !ok {
		s, ok = lookup(l, key+"#other")
	}
	if !ok {
		return key
	}
	return fmt.Sprintf(s, append([]any{n}, args...)...)
}

// pluralCategory follows the CLDR cardinal rules for whole numbers.
func pluralCategory(l Lang, n int) string {
	if n < 0 {
		n = -n
	}
	mod10, mod100 := n%10, n%100
	switch l {
	case "ru", "uk":
		switch {
		case mod10 == 1 && mod100 != 11:
			return "one"
		case mod10 >= 2 && mod10 <= 4 && (mod100 < 12 || mod100 > 14):
			return "few"
		default:
			return "many"
		}
	case "pl":
		switch {
		case n == 1:
			return "one"
		case mod10 >= 2 && mod10 <= 4 && (mod100 < 12 || mod100 > 14):
			return "few"
		default:
			return "many"
		}
	case "cs":
		switch {
		case n == 1:
			return "one"
		case n >= 2 && n <= 4:
			return "few"
		default:
			return "other"
		}
	case "fr":
		if n == 0 || n == 1 {
			return "one"
		}
		return "other"
	default:
		if n == 1 {
			return "one"
		}
		return "other"
	}
}

// Msg is a message decided now and worded later, in the language of whoever
// reads it — a push to someone else is written in theirs, not the sender's.
type Msg struct {
	Key  string
	Args []any
	// Raw is text that is not translated: what a person wrote.
	Raw string
}

// M is the message key with args.
func M(key string, args ...any) Msg { return Msg{Key: key, Args: args} }

// Raw wraps text that is shown as it is.
func Raw(text string) Msg { return Msg{Raw: text} }

// In words the message in l.
func (m Msg) In(l Lang) string {
	if m.Key == "" {
		return m.Raw
	}
	return T(l, m.Key, m.Args...)
}

// IsZero reports whether the message says nothing.
func (m Msg) IsZero() bool { return m.Key == "" && m.Raw == "" }

type ctxKey struct{}

// WithLang carries l in ctx, for code below a handler that has no request.
func WithLang(ctx context.Context, l Lang) context.Context {
	return context.WithValue(ctx, ctxKey{}, l)
}

// FromContext is the language Middleware put in ctx, or Default.
func FromContext(ctx context.Context) Lang {
	if l, ok := ctx.Value(ctxKey{}).(Lang); ok {
		return l
	}
	return Default
}

// Middleware puts the request's language in its context.
func Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		next.ServeHTTP(w, r.WithContext(WithLang(r.Context(), FromRequest(r))))
	})
}
