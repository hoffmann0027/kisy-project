package ws

import (
	"net/http/httptest"
	"testing"
)

// TestOriginChecker locks the fail-closed handshake policy: the WebSocket is
// cookie-authenticated, so a permissive Origin check would allow cross-site
// WebSocket hijacking (CSWSH) from any page a logged-in user visits.
func TestOriginChecker(t *testing.T) {
	cases := []struct {
		name          string
		allowedOrigin string
		origin        string
		host          string
		want          bool
	}{
		// Non-browser clients omit Origin; browsers always send it on
		// cross-site WS handshakes, so this is not a CSWSH vector.
		{"no origin header", "", "", "kisy.example", true},
		{"no origin header with configured origin", "https://kisy.example", "", "kisy.example", true},

		// Same-origin requests pass with or without configuration.
		{"same host, unconfigured", "", "https://kisy.example", "kisy.example", true},
		{"same host, configured", "https://kisy.example", "https://kisy.example", "kisy.example", true},
		{"same host with port (vite dev)", "", "http://localhost:5173", "localhost:5173", true},

		// The regression this test exists for: an unconfigured allowed
		// origin must NOT admit arbitrary origins (fail-open → CSWSH).
		{"foreign origin, unconfigured", "", "https://evil.example", "kisy.example", false},
		{"foreign origin, configured", "https://kisy.example", "https://evil.example", "kisy.example", false},

		// Explicit allowlist match works even when Host differs (edge
		// proxy rewrote it).
		{"configured origin, host rewritten", "https://kisy.example", "https://kisy.example", "backend:8080", true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			check := originChecker(tc.allowedOrigin)
			r := httptest.NewRequest("GET", "/ws", nil)
			r.Host = tc.host
			if tc.origin != "" {
				r.Header.Set("Origin", tc.origin)
			}
			if got := check(r); got != tc.want {
				t.Fatalf("originChecker(%q) origin=%q host=%q: got %v, want %v",
					tc.allowedOrigin, tc.origin, tc.host, got, tc.want)
			}
		})
	}
}
