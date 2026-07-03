package security

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func run(t *testing.T, allowedOrigin, method, host, origin, referer string) int {
	t.Helper()
	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })
	h := CSRF(allowedOrigin)(next)

	req := httptest.NewRequest(method, "http://"+host+"/api/v1/messages", nil)
	req.Host = host
	if origin != "" {
		req.Header.Set("Origin", origin)
	}
	if referer != "" {
		req.Header.Set("Referer", referer)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec.Code
}

func TestCSRFSafeMethodAlwaysAllowed(t *testing.T) {
	if code := run(t, "", http.MethodGet, "kisy.local", "https://evil.example", ""); code != http.StatusOK {
		t.Fatalf("GET should pass regardless of origin, got %d", code)
	}
}

func TestCSRFSameOriginAllowed(t *testing.T) {
	if code := run(t, "", http.MethodPost, "kisy.local", "http://kisy.local", ""); code != http.StatusOK {
		t.Fatalf("same-origin POST should pass, got %d", code)
	}
}

func TestCSRFCrossOriginRejected(t *testing.T) {
	if code := run(t, "", http.MethodPost, "kisy.local", "https://evil.example", ""); code != http.StatusForbidden {
		t.Fatalf("cross-origin POST should be 403, got %d", code)
	}
}

func TestCSRFNoOriginAllowed(t *testing.T) {
	// Non-browser API client: no Origin/Referer → allowed.
	if code := run(t, "", http.MethodPost, "kisy.local", "", ""); code != http.StatusOK {
		t.Fatalf("POST without origin should pass, got %d", code)
	}
}

func TestCSRFRefererFallback(t *testing.T) {
	if code := run(t, "", http.MethodDelete, "kisy.local", "", "https://evil.example/page"); code != http.StatusForbidden {
		t.Fatalf("cross-origin via Referer should be 403, got %d", code)
	}
	if code := run(t, "", http.MethodDelete, "kisy.local", "", "http://kisy.local/page"); code != http.StatusOK {
		t.Fatalf("same-origin via Referer should pass, got %d", code)
	}
}

func TestCSRFConfiguredOrigin(t *testing.T) {
	if code := run(t, "https://app.kisy.example", http.MethodPost, "backend.internal", "https://app.kisy.example", ""); code != http.StatusOK {
		t.Fatalf("explicitly allowed origin should pass, got %d", code)
	}
}

func TestHeadersSet(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })
	rec := httptest.NewRecorder()
	Headers(next).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))

	for _, hdr := range []string{
		"Content-Security-Policy", "X-Content-Type-Options", "X-Frame-Options",
		"Referrer-Policy", "Permissions-Policy", "Strict-Transport-Security",
	} {
		if rec.Header().Get(hdr) == "" {
			t.Errorf("missing security header %s", hdr)
		}
	}
	if got := rec.Header().Get("X-Content-Type-Options"); got != "nosniff" {
		t.Errorf("X-Content-Type-Options = %q, want nosniff", got)
	}
}
