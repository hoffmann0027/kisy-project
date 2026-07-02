package ws

import (
	"net/http"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

// Authenticated is the identity resolved from a WebSocket handshake.
type Authenticated struct {
	UserID    uuid.UUID
	SessionID uuid.UUID
	RoleLevel int
}

// Authenticator validates a handshake request (access cookie or
// access_token query parameter) and returns the actor. Injected to keep
// the ws package independent of the auth package.
type Authenticator func(r *http.Request) (Authenticated, bool)

// Handler upgrades authenticated HTTP requests to WebSocket connections.
type Handler struct {
	hub      *Hub
	auth     Authenticator
	upgrader websocket.Upgrader
}

func NewHandler(hub *Hub, auth Authenticator, allowedOrigin string) *Handler {
	return &Handler{
		hub:  hub,
		auth: auth,
		upgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			CheckOrigin:     originChecker(allowedOrigin),
		},
	}
}

// ServeHTTP authenticates then upgrades the connection. Authentication
// happens before the upgrade so failures return a normal HTTP status.
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	actor, ok := h.auth(r)
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	conn, err := h.upgrader.Upgrade(w, r, nil)
	if err != nil {
		return // Upgrade already wrote the error response
	}

	client := &Client{
		hub:       h.hub,
		conn:      conn,
		send:      make(chan []byte, sendBuffer),
		userID:    actor.UserID,
		sessionID: actor.SessionID,
		roleLevel: actor.RoleLevel,
		subs:      make(map[uuid.UUID]struct{}),
	}

	h.hub.addClient(client)

	go client.writePump()
	go client.readPump()
}

// originChecker permits same-origin handshakes. When allowedOrigin is set
// (production) only that origin is accepted; when empty (development) any
// origin is allowed. Requests without an Origin header (non-browser
// clients) are always allowed.
func originChecker(allowedOrigin string) func(*http.Request) bool {
	return func(r *http.Request) bool {
		origin := r.Header.Get("Origin")
		if origin == "" {
			return true
		}
		if allowedOrigin == "" {
			return true
		}
		return origin == allowedOrigin
	}
}
