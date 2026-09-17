package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"kisy-backend/pkg/httpresponse"
)

// slowQuery behaves like a handler over pgx: it waits for its "query" or for
// the request context, and on cancellation writes its usual error.
func slowQuery(takes time.Duration, errStatus int) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-time.After(takes):
			httpresponse.OK(w, r, http.StatusOK, map[string]string{"results": "ok"})
		case <-r.Context().Done():
			httpresponse.Fail(w, r, errStatus, "WHATEVER", "query failed")
		}
	})
}

func serve(h http.Handler) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/search?q=x", nil))
	return rec
}

func TestExpensiveRequestPastItsDeadlineIs503(t *testing.T) {
	mw := withDeadline(50 * time.Millisecond)
	for _, status := range []int{http.StatusInternalServerError, http.StatusNotFound} {
		start := time.Now()
		rec := serve(mw(slowQuery(5*time.Second, status)))
		if elapsed := time.Since(start); elapsed > time.Second {
			t.Fatalf("the request ran %v past a 50ms deadline", elapsed)
		}
		if rec.Code != http.StatusServiceUnavailable {
			t.Fatalf("handler wrote %d after the deadline: got %d, want 503", status, rec.Code)
		}
		if rec.Header().Get("Retry-After") == "" {
			t.Fatal("503 without Retry-After")
		}
		var env httpresponse.Envelope
		if err := json.Unmarshal(rec.Body.Bytes(), &env); err != nil || env.Success {
			t.Fatalf("body is not one error envelope: %q", rec.Body.String())
		}
	}
}

// A handler that ignores cancellation and writes nothing still yields 503.
func TestSilentHandlerPastItsDeadlineIs503(t *testing.T) {
	rec := serve(withDeadline(30 * time.Millisecond)(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		time.Sleep(80 * time.Millisecond)
	})))
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("got %d, want 503", rec.Code)
	}
}

func TestFastRequestIsUntouched(t *testing.T) {
	rec := serve(withDeadline(time.Second)(slowQuery(time.Millisecond, http.StatusInternalServerError)))
	if rec.Code != http.StatusOK || rec.Header().Get("Retry-After") != "" {
		t.Fatalf("got %d", rec.Code)
	}
	// A genuine client error in time stays what it is.
	rec = serve(withDeadline(time.Second)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		httpresponse.Fail(w, r, http.StatusBadRequest, httpresponse.ErrValidationFailed, "bad")
	})))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("in-time 400 became %d", rec.Code)
	}
}
