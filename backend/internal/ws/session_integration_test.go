//go:build integration

package ws

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/redis/go-redis/v9"

	"kisy-backend/internal/messages"
)

// Audit A-03: a WebSocket was authenticated once, at the handshake, and then
// lived as long as the TCP connection. Logout everywhere, a password change or
// a deactivation revoked the session rows, REST answered 401 — and the socket
// kept delivering new messages and accepting message.send with the old level.
//
// Run with a disposable Redis:
//
//	TEST_REDIS_URL=redis://localhost:6379/0 go test -tags integration ./internal/ws/

type sessionTable struct {
	mu      sync.Mutex
	revoked map[uuid.UUID]bool
}

func (s *sessionTable) revoke(id uuid.UUID) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.revoked[id] = true
}

func (s *sessionTable) live(_ context.Context, _, sessionID uuid.UUID) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return !s.revoked[sessionID], nil
}

type countingSender struct{ n atomic.Int32 }

func (c *countingSender) Send(context.Context, messages.SendInput, messages.ActorMeta) (messages.DTO, error) {
	c.n.Add(1)
	return messages.DTO{}, nil
}

type wsHarness struct {
	hub      *Hub
	sessions *sessionTable
	sender   *countingSender
	url      string
	user     uuid.UUID
}

func newWSHarness(t *testing.T, recheck time.Duration) *wsHarness {
	t.Helper()
	redisURL := os.Getenv("TEST_REDIS_URL")
	if redisURL == "" {
		t.Skip("TEST_REDIS_URL not set; skipping integration test")
	}
	opts, err := redis.ParseURL(redisURL)
	if err != nil {
		t.Fatal(err)
	}
	rdb := redis.NewClient(opts)
	t.Cleanup(func() { _ = rdb.Close() })

	h := &wsHarness{
		sessions: &sessionTable{revoked: map[uuid.UUID]bool{}},
		sender:   &countingSender{},
		user:     uuid.New(),
	}
	h.hub = NewHub(slog.New(slog.NewTextHandler(io.Discard, nil)), rdb,
		func(context.Context, string, uuid.UUID) ([]uuid.UUID, error) { return nil, nil })
	h.hub.SetHandlers(h.sender, nil, nil)
	h.hub.SetSessionChecker(h.sessions.live, recheck)

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	go h.hub.Run(ctx)
	time.Sleep(100 * time.Millisecond) // let the subscription attach

	handler := NewHandler(h.hub, func(r *http.Request) (Authenticated, bool) {
		sid, err := uuid.Parse(r.URL.Query().Get("session"))
		if err != nil {
			return Authenticated{}, false
		}
		return Authenticated{UserID: h.user, SessionID: sid, RoleLevel: 5}, true
	}, "")
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	h.url = "ws" + strings.TrimPrefix(srv.URL, "http")
	return h
}

func (h *wsHarness) dial(t *testing.T, session uuid.UUID) *websocket.Conn {
	t.Helper()
	conn, _, err := websocket.DefaultDialer.Dial(h.url+"?session="+session.String(), nil)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	t.Cleanup(func() { _ = conn.Close() })
	// Wait until the hub has registered it, so a kick cannot race the add.
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		h.hub.mu.RLock()
		n := 0
		for c := range h.hub.clients[h.user] {
			if c.sessionID == session {
				n++
			}
		}
		h.hub.mu.RUnlock()
		if n > 0 {
			return conn
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("client never registered")
	return nil
}

// expectEnded reads until the server closes the socket and checks it said why.
func expectEnded(t *testing.T, conn *websocket.Conn, within time.Duration) {
	t.Helper()
	_ = conn.SetReadDeadline(time.Now().Add(within))
	for {
		_, _, err := conn.ReadMessage()
		if err == nil {
			continue
		}
		var ce *websocket.CloseError
		if errors.As(err, &ce) && ce.Code == CloseSessionEnded {
			return
		}
		t.Fatalf("socket was not ended by the server within %v: %v", within, err)
	}
}

// expectOpen checks the server leaves the socket alone for a while.
func expectOpen(t *testing.T, conn *websocket.Conn, for_ time.Duration) {
	t.Helper()
	_ = conn.SetReadDeadline(time.Now().Add(for_))
	for {
		_, _, err := conn.ReadMessage()
		if err == nil {
			continue
		}
		var ne interface{ Timeout() bool }
		if errors.As(err, &ne) && ne.Timeout() {
			return
		}
		t.Fatalf("a socket of a live session was closed: %v", err)
	}
}

func TestKickSessionEndsThatSessionsSocketOnly(t *testing.T) {
	h := newWSHarness(t, time.Hour)
	s1, s2 := uuid.New(), uuid.New()
	c1, c2 := h.dial(t, s1), h.dial(t, s2)

	h.sessions.revoke(s1)
	h.hub.KickSession(h.user, s1)

	expectEnded(t, c1, 2*time.Second)
	expectOpen(t, c2, 300*time.Millisecond)
}

func TestKickUserKeepsTheCurrentSession(t *testing.T) {
	h := newWSHarness(t, time.Hour)
	current, other1, other2 := uuid.New(), uuid.New(), uuid.New()
	cur, o1, o2 := h.dial(t, current), h.dial(t, other1), h.dial(t, other2)

	// A password change: every other session goes.
	h.hub.KickUser(h.user, current)

	expectEnded(t, o1, 2*time.Second)
	expectEnded(t, o2, 2*time.Second)
	expectOpen(t, cur, 300*time.Millisecond)
}

func TestKickUserWithoutKeepEndsEverySocket(t *testing.T) {
	h := newWSHarness(t, time.Hour)
	a, b := h.dial(t, uuid.New()), h.dial(t, uuid.New())

	// Deactivation, logout everywhere, role change.
	h.hub.KickUser(h.user, uuid.Nil)

	expectEnded(t, a, 2*time.Second)
	expectEnded(t, b, 2*time.Second)
}

// A revocation the kick never announces — cmd/resetpw writes the sessions
// table directly, or the publish was lost — still ends the socket.
func TestRevokedSessionEndsWithoutAKick(t *testing.T) {
	h := newWSHarness(t, 150*time.Millisecond)
	s := uuid.New()
	conn := h.dial(t, s)

	// Several re-checks pass while the session is live. (Checked on the server
	// side: a timed-out read would poison the client connection.)
	time.Sleep(400 * time.Millisecond)
	h.hub.mu.RLock()
	registered := len(h.hub.clients[h.user])
	h.hub.mu.RUnlock()
	if registered != 1 {
		t.Fatalf("a live session's socket was dropped by its re-check (%d registered)", registered)
	}
	h.sessions.revoke(s)
	expectEnded(t, conn, 2*time.Second)
}

// A frame sent on a socket whose session was revoked is not acted on, even
// before the periodic re-check comes round.
func TestFrameFromRevokedSessionIsRefused(t *testing.T) {
	h := newWSHarness(t, time.Hour)
	s := uuid.New()
	conn := h.dial(t, s)

	send := `{"type":"message.send","data":{"chatType":"group","chatId":"` + uuid.NewString() + `","text":"after revoke"}}`
	if err := conn.WriteMessage(websocket.TextMessage, []byte(send)); err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(2 * time.Second)
	for h.sender.n.Load() == 0 && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	if h.sender.n.Load() != 1 {
		t.Fatalf("a live session's message.send must reach the sender, got %d", h.sender.n.Load())
	}

	h.sessions.revoke(s)
	// Age the last check past inboundRecheck instead of sleeping through it.
	h.hub.mu.RLock()
	for c := range h.hub.clients[h.user] {
		c.checkedAt.Store(0)
	}
	h.hub.mu.RUnlock()

	if err := conn.WriteMessage(websocket.TextMessage, []byte(send)); err != nil {
		t.Fatal(err)
	}
	expectEnded(t, conn, 2*time.Second)
	if n := h.sender.n.Load(); n != 1 {
		t.Fatalf("message.send from a revoked session was acted on (%d sends)", n)
	}
}
