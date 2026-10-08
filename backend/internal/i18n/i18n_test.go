package i18n

import (
	"net/http/httptest"
	"reflect"
	"regexp"
	"strings"
	"testing"
)

func TestTheLanguageComesFromTheAppNotTheDevice(t *testing.T) {
	r := httptest.NewRequest("GET", "/", nil)
	r.Header.Set("Accept-Language", "en-US,en;q=0.9")
	if got := FromRequest(r); got != "ru" {
		t.Fatalf("an app that names no language is a Russian one: got %q", got)
	}
	r.Header.Set(Header, "en")
	if got := FromRequest(r); got != "en" {
		t.Fatalf("got %q, want en", got)
	}
	r.Header.Set(Header, "klingon")
	if got := FromRequest(r); got != Default {
		t.Fatalf("an unknown language falls back to %q, got %q", Default, got)
	}
}

func TestParse(t *testing.T) {
	for tag, want := range map[string]Lang{"en-US": "en", "EN": "en", "ru_RU": "ru", "": "ru", "xx": "ru"} {
		if got := Parse(tag); got != want {
			t.Errorf("Parse(%q) = %q, want %q", tag, got, want)
		}
	}
}

func TestCountablePhrasesFollowEachLanguagesRules(t *testing.T) {
	cases := []struct {
		lang Lang
		n    int
		want string
	}{
		{"ru", 1, "через 1 час"},
		{"ru", 3, "через 3 часа"},
		{"ru", 5, "через 5 часов"},
		{"ru", 11, "через 11 часов"},
		{"ru", 21, "через 21 час"},
		{"ru", 22, "через 22 часа"},
		{"en", 1, "in 1 hour"},
		{"en", 5, "in 5 hours"},
	}
	for _, c := range cases {
		if got := N(c.lang, "quarantine.inHours", c.n); got != c.want {
			t.Errorf("N(%s, %d) = %q, want %q", c.lang, c.n, got, c.want)
		}
	}
	for _, c := range []struct {
		lang Lang
		n    int
		want string
	}{{"pl", 22, "few"}, {"pl", 25, "many"}, {"cs", 3, "few"}, {"cs", 5, "other"}, {"fr", 0, "one"}, {"uk", 12, "many"}} {
		if got := pluralCategory(c.lang, c.n); got != c.want {
			t.Errorf("pluralCategory(%s, %d) = %q, want %q", c.lang, c.n, got, c.want)
		}
	}
}

func TestAMessageIsWordedForWhoeverReadsIt(t *testing.T) {
	m := M("releases.pushTitle", "2.0")
	if got := m.In("ru"); got != "Вышла версия 2.0" {
		t.Fatalf("ru: %q", got)
	}
	if got := m.In("en"); got != "Version 2.0 is out" {
		t.Fatalf("en: %q", got)
	}
	if got := Raw("как есть").In("en"); got != "как есть" {
		t.Fatalf("raw text is not translated: %q", got)
	}
}

var verbs = regexp.MustCompile(`%[-+# 0]*\d*(?:\.\d+)?[a-zA-Z%]`)

func baseKey(k string) string {
	if i := strings.IndexByte(k, '#'); i >= 0 {
		return k[:i]
	}
	return k
}

// Every language says everything Russian says, with the same fmt verbs in the
// same order — a dropped %d prints "%!d(MISSING)" on someone's screen.
func TestEveryCatalogHasEveryMessage(t *testing.T) {
	for _, l := range Supported {
		cat, ok := catalogs[l]
		if !ok {
			t.Fatalf("%s: no catalog", l)
		}
		for key, source := range ru {
			if strings.Contains(key, "#") {
				if _, ok := cat[baseKey(key)+"#other"]; !ok {
					t.Errorf("%s: %s#other missing", l, baseKey(key))
				}
				continue
			}
			msg, ok := cat[key]
			if !ok {
				t.Errorf("%s: %s missing", l, key)
				continue
			}
			if want, got := verbs.FindAllString(source, -1), verbs.FindAllString(msg, -1); !reflect.DeepEqual(want, got) {
				t.Errorf("%s: %s has verbs %v, Russian has %v", l, key, got, want)
			}
		}
		for key := range cat {
			if _, ok := ru[key]; !ok && !strings.Contains(key, "#") {
				t.Errorf("%s: %s is not a Russian key", l, key)
			}
		}
	}
}
