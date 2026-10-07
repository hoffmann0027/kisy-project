package metrics

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
)

// Audit A-18: the method label was the raw request method, which the client
// chooses freely. Each invented method minted a fresh set of series kept in
// memory for good, so a stream of made-up methods grew the process without
// limit — no account needed.
func TestInventedMethodsDoNotMintNewSeries(t *testing.T) {
	h := Middleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	before := seriesCount(httpRequestsTotal)
	for _, m := range []string{"AUDITXA", "AUDITXB", "AUDITXC", "BREW", "PROPFIND"} {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest(m, "/health", nil))
	}
	grown := seriesCount(httpRequestsTotal) - before
	if grown > 1 {
		t.Fatalf("five invented methods created %d new series; they must all share one (OTHER)", grown)
	}

	for _, m := range []string{"AUDITXA", "TRACE", "CONNECT"} {
		if got := MethodLabel(m); got != "OTHER" {
			t.Errorf("MethodLabel(%q) = %q, want OTHER", m, got)
		}
	}
	for _, m := range []string{http.MethodGet, http.MethodPost, http.MethodDelete} {
		if got := MethodLabel(m); got != m {
			t.Errorf("MethodLabel(%q) = %q, want it kept", m, got)
		}
	}
	if strings.Contains(MethodLabel("get"), "get") {
		t.Error("a lower-case method is not a standard method and must not get its own label")
	}
}

// seriesCount is how many series a collector holds right now.
func seriesCount(c prometheus.Collector) int {
	ch := make(chan prometheus.Metric, 4096)
	c.Collect(ch)
	close(ch)
	n := 0
	for range ch {
		n++
	}
	return n
}
