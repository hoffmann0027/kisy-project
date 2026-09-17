package security

import (
	"net/http"
	"net/http/httptest"
	"os"
	"regexp"
	"strings"
	"testing"
)

// directives splits a policy into directive → sources.
func directives(policy string) map[string]string {
	out := map[string]string{}
	for _, d := range strings.Split(policy, ";") {
		fields := strings.Fields(d)
		if len(fields) > 0 {
			out[fields[0]] = strings.Join(fields[1:], " ")
		}
	}
	return out
}

// The sign-up screen loads the Turnstile widget: its script and its iframe
// come from challenges.cloudflare.com. Without both the widget never renders
// and nobody can register.
func TestCSPAllowsTheTurnstileWidget(t *testing.T) {
	rec := httptest.NewRecorder()
	Headers(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {})).
		ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	d := directives(rec.Header().Get("Content-Security-Policy"))
	for _, name := range []string{"script-src", "frame-src"} {
		if !strings.Contains(d[name], "https://challenges.cloudflare.com") {
			t.Errorf("%s = %q, want challenges.cloudflare.com", name, d[name])
		}
	}
	// Nothing else is widened.
	if d["script-src"] != "'self' https://challenges.cloudflare.com" {
		t.Errorf("script-src = %q", d["script-src"])
	}
	if d["frame-ancestors"] != "'none'" {
		t.Errorf("frame-ancestors = %q", d["frame-ancestors"])
	}
}

// Nginx sets the same header at the edge; the two must not drift, or the
// widget works in one deployment shape and not the other.
func TestNginxCSPMatchesBackend(t *testing.T) {
	re := regexp.MustCompile(`add_header Content-Security-Policy "([^"]+)"`)
	for _, f := range []string{"../../../../deploy/nginx/nginx.conf", "../../../../deploy/nginx/nginx.tls.conf"} {
		raw, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		m := re.FindAllSubmatch(raw, -1)
		if len(m) == 0 {
			t.Fatalf("%s: no CSP header", f)
		}
		for _, got := range m {
			if string(got[1]) != contentSecurityPolicy {
				t.Errorf("%s:\n nginx   %s\n backend %s", f, got[1], contentSecurityPolicy)
			}
		}
	}
}
