package ws

import (
	"net/http"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"

	"kisy-backend/internal/platform/security"
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

// originChecker permits handshakes from the configured allowed origin or from
// the same origin as the request (Origin host == request Host, which the edge
// proxy and the Vite dev proxy both preserve). It is fail-closed: an
// unconfigured allowedOrigin no longer admits arbitrary origins — cookies
// authenticate the handshake, so a permissive check would allow cross-site
// WebSocket hijacking (CSWSH) from any page the user visits. Requests without
// an Origin header (non-browser clients) are allowed: browsers always send
// Origin on cross-site WebSocket handshakes, so those cannot be CSWSH.
func originChecker(allowedOrigin string) func(*http.Request) bool {
	return func(r *http.Request) bool {
		origin := r.Header.Get("Origin")
		if origin == "" {
			return true
		}
		return security.OriginAllowed(origin, r, allowedOrigin)
	}
}
