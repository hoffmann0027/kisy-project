//go:build integration

package ws

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

// Connection caps: 3 sockets per account and 10 per address by default, the
// oldest closed when a new one goes over; and a flood of frames closes the
// socket that sent it.

type capsHarness struct {
	hub *Hub
	url string
}

func newCapsHarness(t *testing.T, limits ConnLimits) capsHarness {
	t.Helper()
	base := newWSHarness(t, time.Hour) // Redis, session checker, running hub
	base.hub.SetConnLimits(limits)
	// Every test socket comes from 127.0.0.1; start from an empty count so a
	// previous run's leftovers cannot shift which socket is oldest.
	base.hub.rdb.Del(context.Background(), connKeyPrefix+"ip:127.0.0.1")

	handler := NewHandler(base.hub, func(r *http.Request) (Authenticated, bool) {
		user, err := uuid.Parse(r.URL.Query().Get("user"))
		if err != nil {
			return Authenticated{}, false
		}
		return Authenticated{UserID: user, SessionID: uuid.New(), RoleLevel: 5}, true
	}, "")
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	return capsHarness{hub: base.hub, url: "ws" + strings.TrimPrefix(srv.URL, "http")}
}

func (h capsHarness) dial(t *testing.T, user uuid.UUID) *websocket.Conn {
	t.Helper()
	conn, _, err := websocket.DefaultDialer.Dial(h.url+"?user="+user.String(), nil)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	t.Cleanup(func() { _ = conn.Close() })
	// Registration (and any eviction it causes) happens in the handshake
	// goroutine; give it a moment so the next dial is strictly newer.
	time.Sleep(30 * time.Millisecond)
	return conn
}

func expectClosedWith(t *testing.T, conn *websocket.Conn, code int, within time.Duration) {
	t.Helper()
	_ = conn.SetReadDeadline(time.Now().Add(within))
	for {
		_, _, err := conn.ReadMessage()
		if err == nil {
			continue
		}
		var ce *websocket.CloseError
		if errors.As(err, &ce) && ce.Code == code {
			return
		}
		t.Fatalf("want close %d within %v, got %v", code, within, err)
	}
}

func TestFourthSocketOfAnAccountClosesTheOldest(t *testing.T) {
	h := newCapsHarness(t, ConnLimits{MaxPerUser: 3})
	user := uuid.New()
	oldest := h.dial(t, user)
	second, third := h.dial(t, user), h.dial(t, user)
	newest := h.dial(t, user)

	expectClosedWith(t, oldest, CloseReplaced, 2*time.Second)
	for _, c := range []*websocket.Conn{second, third, newest} {
		expectOpen(t, c, 200*time.Millisecond)
	}
}

func TestEleventhSocketFromOneAddressClosesTheOldest(t *testing.T) {
	h := newCapsHarness(t, ConnLimits{MaxPerUser: 3, MaxPerIP: 10})
	var conns []*websocket.Conn
	for i := 0; i < 11; i++ {
		// Different accounts: only the per-address cap applies.
		conns = append(conns, h.dial(t, uuid.New()))
	}
	expectClosedWith(t, conns[0], CloseReplaced, 2*time.Second)
	for _, c := range conns[1:] {
		expectOpen(t, c, 100*time.Millisecond)
	}
}

// A socket that closes on its own frees its place: no one is evicted.
func TestClosedSocketsDoNotCount(t *testing.T) {
	h := newCapsHarness(t, ConnLimits{MaxPerUser: 2})
	user := uuid.New()
	a := h.dial(t, user)
	b := h.dial(t, user)
	_ = b.Close()
	time.Sleep(100 * time.Millisecond)
	h.dial(t, user)
	expectOpen(t, a, 300*time.Millisecond)
}

func TestFrameFloodClosesTheSocket(t *testing.T) {
	h := newCapsHarness(t, ConnLimits{MaxFramesPerSecond: 30})
	conn := h.dial(t, uuid.New())
	frame := []byte(`{"type":"typing.start","data":{}}`)
	for i := 0; i < 40; i++ {
		if err := conn.WriteMessage(websocket.TextMessage, frame); err != nil {
			break // the server may already have closed it
		}
	}
	expectClosedWith(t, conn, CloseTooFast, 2*time.Second)

	// Within the rate nothing happens.
	calm := h.dial(t, uuid.New())
	for i := 0; i < 20; i++ {
		if err := calm.WriteMessage(websocket.TextMessage, frame); err != nil {
			t.Fatal(err)
		}
	}
	expectOpen(t, calm, 300*time.Millisecond)
}
