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
	if d["script-src"] != "'self' 'wasm-unsafe-eval' https://challenges.cloudflare.com" {
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
			// nginx writes the host as $host; the backend takes it from the request.
			nginx := strings.ReplaceAll(string(got[1]), "$host", "kisy.example")
			if want := contentSecurityPolicy("kisy.example"); nginx != want {
				t.Errorf("%s:\n nginx   %s\n backend %s", f, nginx, want)
			}
		}
	}
}

// Audit A-45: connect-src allowed "ws: wss:" — a socket to any host, the
// channel an XSS would use to carry the session's messages away.
func TestSocketsOnlyToThisHost(t *testing.T) {
	csp := func(host string) map[string]string {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Host = host
		Headers(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {})).ServeHTTP(rec, req)
		return directives(rec.Header().Get("Content-Security-Policy"))
	}
	if got := csp("kisy.onrender.com")["connect-src"]; got != "'self' ws://kisy.onrender.com wss://kisy.onrender.com" {
		t.Fatalf("connect-src = %q", got)
	}
	if got := csp("localhost:8080")["connect-src"]; got != "'self' ws://localhost:8080 wss://localhost:8080" {
		t.Fatalf("connect-src with a port = %q", got)
	}
	// A Host header cannot add a directive or a source of its own.
	for _, evil := range []string{"a.example; script-src *", "a.example wss:", "*", ""} {
		d := csp(evil)
		if d["connect-src"] != "'self'" || d["script-src"] != "'self' 'wasm-unsafe-eval' https://challenges.cloudflare.com" {
			t.Fatalf("host %q produced connect-src %q, script-src %q", evil, d["connect-src"], d["script-src"])
		}
	}
}

// libsodium is WebAssembly with no fallback: under a script-src without
// 'wasm-unsafe-eval' sodium.ready rejects and no browser can start an E2EE
// session. Only WebAssembly is allowed — never eval.
func TestCSPLetsWebAssemblyCompileButNotEval(t *testing.T) {
	sources := strings.Fields(directives(contentSecurityPolicy("kisy.example"))["script-src"])
	has := func(s string) bool {
		for _, f := range sources {
			if f == s {
				return true
			}
		}
		return false
	}
	if !has("'wasm-unsafe-eval'") {
		t.Fatalf("script-src %v: libsodium cannot start", sources)
	}
	if has("'unsafe-eval'") || has("'unsafe-inline'") {
		t.Fatalf("script-src %v allows eval or inline script", sources)
	}
}

func TestAPIResponsesAreNotStored(t *testing.T) {
	rec := httptest.NewRecorder()
	NoStore(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {})).
		ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/users/me", nil))
	if got := rec.Header().Get("Cache-Control"); got != "no-store" {
		t.Fatalf("Cache-Control = %q", got)
	}
	// A file handler's own caching still wins.
	rec = httptest.NewRecorder()
	NoStore(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Cache-Control", "private, max-age=60")
	})).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/attachments/x", nil))
	if got := rec.Header().Get("Cache-Control"); got != "private, max-age=60" {
		t.Fatalf("a file's Cache-Control was overridden: %q", got)
	}
}
