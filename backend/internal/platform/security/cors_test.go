package security

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
)

const appOrigin = "https://localhost"

// router mirrors how the real server mounts CORS: inside a chi Route that only
// declares POST. Without the middleware a preflight would hit chi's
// MethodNotAllowed and return 405, which makes the WebView drop the request.
func router() http.Handler {
	r := chi.NewRouter()
	r.Route("/api/v1", func(r chi.Router) {
		r.Use(CORS([]string{appOrigin}))
		r.Post("/auth/login", func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		})
	})
	return r
}

func TestPreflightFromAppOriginIsAnswered(t *testing.T) {
	req := httptest.NewRequest(http.MethodOptions, "/api/v1/auth/login", nil)
	req.Header.Set("Origin", appOrigin)
	req.Header.Set("Access-Control-Request-Method", "POST")
	rec := httptest.NewRecorder()

	router().ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("preflight status = %d, want 204", rec.Code)
	}
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != appOrigin {
		t.Fatalf("allow-origin = %q, want %q", got, appOrigin)
	}
	for _, h := range []string{"Authorization", "X-Kisy-Client", "X-File-Name"} {
		if !strings.Contains(rec.Header().Get("Access-Control-Allow-Headers"), h) {
			t.Fatalf("allow-headers missing %q: %q", h, rec.Header().Get("Access-Control-Allow-Headers"))
		}
	}
}

func TestActualRequestGetsAllowOrigin(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", nil)
	req.Header.Set("Origin", appOrigin)
	rec := httptest.NewRecorder()

	router().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != appOrigin {
		t.Fatalf("allow-origin = %q, want %q", got, appOrigin)
	}
	// Credentials must never be allowed: native clients use Bearer tokens.
	if rec.Header().Get("Access-Control-Allow-Credentials") != "" {
		t.Fatal("credentialed CORS must not be enabled")
	}
}

func TestForeignOriginGetsNoCORS(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", nil)
	req.Header.Set("Origin", "https://evil.example")
	rec := httptest.NewRecorder()

	router().ServeHTTP(rec, req)

	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Fatalf("an unlisted origin must get no allow-origin, got %q", got)
	}
}

// The web app is same-origin and sends no Origin on its own requests.
func TestSameOriginRequestUntouched(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", nil)
	rec := httptest.NewRecorder()

	router().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	if rec.Header().Get("Access-Control-Allow-Origin") != "" {
		t.Fatal("same-origin response must not carry CORS headers")
	}
}

func contains(haystack, needle string) bool {
	return len(haystack) > 0 && len(needle) > 0 && (haystack == needle ||
		len(haystack) >= len(needle) && (indexOf(haystack, needle) >= 0))
}

func indexOf(h, n string) int {
	for i := 0; i+len(n) <= len(h); i++ {
		if h[i:i+len(n)] == n {
			return i
		}
	}
	return -1
}
