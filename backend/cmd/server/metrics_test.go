package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// Audit A-18: /metrics was protected by the belief that a proxy hides it. The
// managed deploy has no proxy — the backend serves the SPA itself — so
// https://<host>/metrics answered the public internet with route-level request
// counts, database pool state and Go runtime detail. Verified live on
// 28.09.2026: HTTP 200 from the open internet.
func TestMetricsEndpointIsClosedUnlessTheScraperProvesItself(t *testing.T) {
	const token = "scrape-token-value"

	cases := []struct {
		name       string
		token      string
		open       bool
		authHeader string
		want       int
	}{
		{"production without a token configured", "", false, "", http.StatusNotFound},
		{"production, token configured, no header", token, false, "", http.StatusNotFound},
		{"production, token configured, wrong token", token, false, "Bearer nope", http.StatusNotFound},
		{"production, token configured, right token", token, false, "Bearer " + token, http.StatusOK},
		{"development, no token", "", true, "", http.StatusOK},
		// A token set in development is still a token: it must be honoured,
		// otherwise "open" would be a way around it.
		{"development, token configured, wrong token", token, true, "Bearer nope", http.StatusNotFound},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
			if tc.authHeader != "" {
				req.Header.Set("Authorization", tc.authHeader)
			}
			rec := httptest.NewRecorder()
			metricsHandler(tc.token, tc.open)(rec, req)

			if rec.Code != tc.want {
				t.Fatalf("status = %d, want %d", rec.Code, tc.want)
			}
			if tc.want == http.StatusOK && rec.Body.Len() == 0 {
				t.Fatal("an authorised scrape got an empty body")
			}
		})
	}
}
