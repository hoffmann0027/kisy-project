package ws

import (
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

const (
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	pingPeriod     = (pongWait * 9) / 10
	maxMessageSize = 16 * 1024
	sendBuffer     = 32

	// CloseSessionEnded is the close code a client gets when the session its
	// socket authenticated with was revoked (logout, password change,
	// deactivation, role change). A reconnect with the same token is refused,
	// so the client has to renew or sign in again.
	CloseSessionEnded = 4001

	// sessionRecheck bounds how long a socket can outlive its session when
	// the instant kick never arrives — a Redis outage, or a session revoked
	// outside the services (cmd/resetpw writes the table directly).
	sessionRecheck = 30 * time.Second
	// inboundRecheck is how stale a session check may be before a frame the
	// client sends — a message, a call signal — is acted on.
	inboundRecheck = 5 * time.Second
)

// Client is one WebSocket connection for one authenticated user. A user
// may have several clients (devices/tabs).
type Client struct {
	hub       *Hub
	conn      *websocket.Conn
	send      chan []byte
	userID    uuid.UUID
	sessionID uuid.UUID
	roleLevel int

	// subs is the set of user IDs whose presence this client wants.
	subs map[uuid.UUID]struct{}

	// done is closed when the client is deregistered; it stops watchSession.
	done    chan struct{}
	endOnce sync.Once
	// checkedAt is when the session was last confirmed live (unix nanos).
	checkedAt atomic.Int64

	// connID, connectedAt and ipBucket identify the socket to the connection
	// caps (connlimit.go); ipBucket is the client address, IPv6 as its /64.
	connID      uuid.UUID
	connectedAt time.Time
	ipBucket    string
}

// end closes the connection because its session is over. The close frame
// tells the client why; closing the socket makes readPump return, which
// deregisters the client. Safe from any goroutine, any number of times
// (gorilla allows WriteControl and Close concurrently with the pumps).
func (c *Client) end(reason string) { c.closeWith(CloseSessionEnded, reason) }

// closeWith closes the connection with the given close code. Only the first
// call has an effect.
func (c *Client) closeWith(code int, reason string) {
	c.endOnce.Do(func() {
		msg := websocket.FormatCloseMessage(code, reason)
		_ = c.conn.WriteControl(websocket.CloseMessage, msg, time.Now().Add(writeWait))
		_ = c.conn.Close()
	})
}

// watchSession re-checks the session on a timer and ends the connection once
// it is revoked. The kick published on revocation is the fast path; this is
// the one that cannot be missed.
func (c *Client) watchSession(every time.Duration) {
	t := time.NewTicker(every)
	defer t.Stop()
	for {
		select {
		case <-c.done:
			return
		case <-t.C:
			if !c.hub.sessionLive(c) {
				c.end("session ended")
				return
			}
		}
	}
}

// readPump reads inbound frames until the connection closes, dispatching
// each to the hub's handler. It owns deregistration on exit.
func (c *Client) readPump() {
	defer func() {
		c.hub.removeClient(c)
		_ = c.conn.Close()
	}()

	c.conn.SetReadLimit(maxMessageSize)
	_ = c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		return c.conn.SetReadDeadline(time.Now().Add(pongWait))
	})

	// Inbound frames per second, counted over whole-second windows. Past the
	// cap the socket is closed: a client that floods frames is broken or
	// hostile, and each frame costs a session check and often a database call.
	maxFrames := c.hub.limits.MaxFramesPerSecond
	windowStart, frames := time.Now(), 0
	for {
		_, raw, err := c.conn.ReadMessage()
		if err != nil {
			return
		}
		if maxFrames > 0 {
			if now := time.Now(); now.Sub(windowStart) >= time.Second {
				windowStart, frames = now, 0
			}
			frames++
			if frames > maxFrames {
				c.closeWith(CloseTooFast, "too many frames")
				return
			}
		}
		c.hub.handleInbound(c, raw)
	}
}

// writePump drains the send channel and keeps the connection warm with
// pings. It is the sole writer to the connection.
func (c *Client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		_ = c.conn.Close()
	}()

	for {
		select {
		case msg, ok := <-c.send:
			_ = c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				_ = c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			if err := c.conn.WriteMessage(websocket.TextMessage, msg); err != nil {
				return
			}
		case <-ticker.C:
			// Keep this user's presence key alive while the socket is up; it
			// expires on its own if the process dies (see presenceTTL).
			c.hub.touchPresence(c.userID)
			c.hub.heartbeatConn(c)
			_ = c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// enqueue tries to send a frame to this client, dropping it if the client's
// buffer is full (a slow consumer must not block the hub).
func (c *Client) enqueue(msg []byte) {
	select {
	case c.send <- msg:
	default:
	}
}
