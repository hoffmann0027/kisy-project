package dsn

import (
	"errors"
	"fmt"
	"net/url"
	"strings"
	"testing"
)

func TestScrubMasksThePasswordInEveryForm(t *testing.T) {
	cases := []struct{ raw, password string }{
		// The audit's reproduction: a URL net/url refuses, quoted whole.
		{"postgres://kisy:SuperSecretPW1@host:notaport/kisy", "SuperSecretPW1"},
		{"pgx5://kisy:SuperSecretPW1@host:notaport/kisy", "SuperSecretPW1"},
		{"redis://:SuperSecretPW1@host:notaport", "SuperSecretPW1"},
		// A valid URL, its password percent-encoded.
		{"postgres://kisy:p%40ss%2Fword@db.example:5432/kisy", "p@ss/word"},
		// A keyword DSN, plain and quoted.
		{"host=db user=kisy password=SuperSecretPW1 dbname=kisy", "SuperSecretPW1"},
		{"host=db user=kisy password='Super Secret' dbname=kisy", "Super Secret"},
	}
	for _, c := range cases {
		_, parseErr := url.Parse(c.raw)
		original := fmt.Errorf("connect %q (%s): %v", c.raw, c.password, parseErr)
		got := Scrub(original, c.raw).Error()
		if strings.Contains(got, c.password) {
			t.Errorf("%s: password left in %q", c.raw, got)
		}
		if enc := url.QueryEscape(c.password); strings.Contains(got, enc) {
			t.Errorf("%s: encoded password left in %q", c.raw, got)
		}
	}
}

func TestScrubKeepsTheRestAndNil(t *testing.T) {
	if Scrub(nil, "postgres://a:b@c/d") != nil {
		t.Fatal("a nil error became non-nil")
	}
	err := Scrub(errors.New(`dial tcp db.example:5432: connection refused`), "postgres://kisy:pw@db.example:5432/kisy")
	if err.Error() != "dial tcp db.example:5432: connection refused" {
		t.Fatalf("an error without the password was changed: %q", err)
	}
}
