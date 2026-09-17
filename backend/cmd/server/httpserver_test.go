package main

import (
	"bytes"
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// Audit B-10: every timeout is set, and they actually reach the server.
func TestServerTimeoutsAreSet(t *testing.T) {
	srv := newHTTPServer(8080, http.NotFoundHandler(), defaultServerTimeouts)
	if srv.ReadHeaderTimeout <= 0 || srv.ReadTimeout <= 0 || srv.IdleTimeout <= 0 {
		t.Fatalf("header=%v read=%v idle=%v", srv.ReadHeaderTimeout, srv.ReadTimeout, srv.IdleTimeout)
	}
}

// A client that sends headers and then trickles its body is cut off by
// ReadTimeout instead of holding the connection for as long as it likes.
func TestSlowBodyIsCutOff(t *testing.T) {
	gotErr := make(chan error, 1)
	srv := newHTTPServer(0, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, err := io.ReadAll(r.Body)
		gotErr <- err
	}), serverTimeouts{ReadHeader: time.Second, Read: 300 * time.Millisecond, Idle: time.Second})
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	go func() { _ = srv.Serve(ln) }()
	t.Cleanup(func() { _ = srv.Close() })

	conn, err := net.Dial("tcp", ln.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	_, _ = conn.Write([]byte("POST / HTTP/1.1\r\nHost: x\r\nContent-Length: 1000\r\n\r\nabc"))

	select {
	case err := <-gotErr:
		if err == nil {
			t.Fatal("a body that never completes was read to the end")
		}
	case <-time.After(3 * time.Second):
		t.Fatal("the slow body was still being read after 3s")
	}
}

func bodyLimited(reached *bool, read *error) http.Handler {
	return limitBody(maxJSONBody, isUploadRoute)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		*reached = true
		_, *read = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusOK)
	}))
}

func TestOversizedJSONBodyIsRefused(t *testing.T) {
	big := bytes.Repeat([]byte("a"), maxJSONBody+1)

	// Declared size: refused before any handler runs, authentication included.
	var reached bool
	var readErr error
	rec := httptest.NewRecorder()
	bodyLimited(&reached, &readErr).ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/v1/messages", bytes.NewReader(big)))
	if rec.Code != http.StatusRequestEntityTooLarge || reached {
		t.Fatalf("declared 1 MiB+1: status %d, handler reached %v", rec.Code, reached)
	}

	// Undeclared size (chunked): the read stops at the cap.
	req := httptest.NewRequest(http.MethodPost, "/api/v1/messages", io.NopCloser(bytes.NewReader(big)))
	req.ContentLength = -1
	bodyLimited(&reached, &readErr).ServeHTTP(httptest.NewRecorder(), req)
	var tooBig *http.MaxBytesError
	if !errors.As(readErr, &tooBig) {
		t.Fatalf("chunked 1 MiB+1 read: %v", readErr)
	}

	// Within the cap: untouched.
	reached, readErr = false, nil
	rec = httptest.NewRecorder()
	bodyLimited(&reached, &readErr).ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/api/v1/messages", strings.NewReader(`{"text":"hi"}`)))
	if rec.Code != http.StatusOK || readErr != nil {
		t.Fatalf("small body: %d %v", rec.Code, readErr)
	}
}

// Uploads carry their own, larger, quota-derived caps — and only uploads.
func TestUploadRoutesAreExemptAndNothingElse(t *testing.T) {
	uploads := [][2]string{
		{http.MethodPost, "/api/v1/attachments"},
		{http.MethodPut, "/api/v1/attachments/0b9f6c1e-1111-4222-8333-944445555666/chunk"},
		{http.MethodPost, "/api/v1/notes/file"},
		{http.MethodPost, "/api/v1/posts/0b9f6c1e-1111-4222-8333-944445555666/media"},
		{http.MethodPost, "/api/v1/users/me/avatar"},
		{http.MethodPost, "/api/v1/groups/0b9f6c1e-1111-4222-8333-944445555666/avatar"},
	}
	for _, u := range uploads {
		var reached bool
		var readErr error
		big := bytes.Repeat([]byte("a"), maxJSONBody+1)
		rec := httptest.NewRecorder()
		bodyLimited(&reached, &readErr).ServeHTTP(rec, httptest.NewRequest(u[0], u[1], bytes.NewReader(big)))
		if rec.Code != http.StatusOK || readErr != nil {
			t.Errorf("%s %s: %d %v", u[0], u[1], rec.Code, readErr)
		}
	}
	for _, not := range [][2]string{
		{http.MethodPost, "/api/v1/messages"},
		{http.MethodPost, "/api/v1/attachments/init"},
		{http.MethodGet, "/api/v1/attachments"},
		{http.MethodPost, "/api/v1/users/me/avatar/extra"},
		{http.MethodPost, "/api/v1/auth/register"},
	} {
		if isUploadRoute(httptest.NewRequest(not[0], not[1], nil)) {
			t.Errorf("%s %s must not be exempt", not[0], not[1])
		}
	}
}
