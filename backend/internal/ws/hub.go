package ws

import (
	"context"
	"encoding/json"
	"log/slog"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"

	"kisy-backend/internal/messages"
	"kisy-backend/internal/platform/metrics"
)

// Sender persists a message sent over the socket. Satisfied by
// *messages.Service; injected to keep inbound handling in the hub without
// leaking unexported types to the composition root.
type Sender interface {
	Send(ctx context.Context, in messages.SendInput, actor messages.ActorMeta) (messages.DTO, error)
}

// ChatAuthorizer reports whether an actor may emit real-time events
// (typing, read) into a chat. Returns a non-nil error to deny.
type ChatAuthorizer func(ctx context.Context, chatType string, chatID, actorID uuid.UUID, actorLevel int) error

// CallActor identifies the user emitting a call-signaling frame.
type CallActor struct {
	UserID    uuid.UUID
	SessionID uuid.UUID
	RoleLevel int
}

// CallSignaler validates and relays a single "call.*" signaling frame. A
// returned error is surfaced to the sender as a generic error event. Satisfied
// structurally by *calls.Service; injected to avoid a ws→calls dependency here.
type CallSignaler interface {
	HandleSignal(ctx context.Context, actor CallActor, msgType string, data json.RawMessage) error
}

// Redis pub/sub channels. Targeted events carry an explicit recipient list;
// presence changes are broadcast and filtered per-instance by local
// subscription state.
const (
	channelFanout    = "kisy:ws:fanout"
	channelPresence  = "kisy:ws:presence"
	channelBroadcast = "kisy:ws:broadcast"
	channelKick      = "kisy:ws:kick"
)

// SessionChecker reports whether a socket's session is still live: not
// revoked, not expired, and still the user's. An error means "unknown".
type SessionChecker func(ctx context.Context, userID, sessionID uuid.UUID) (bool, error)

// RecipientResolver returns the user IDs that should receive events for a
// chat (participants for private chats, members for groups). Injected to
// avoid a ws→chats/groups import cycle.
type RecipientResolver func(ctx context.Context, chatType string, chatID uuid.UUID) ([]uuid.UUID, error)

// Hub owns all local client connections and bridges events across
// instances through Redis pub/sub.
type Hub struct {
	log     *slog.Logger
	rdb     *redis.Client
	resolve RecipientResolver

	mu          sync.RWMutex
	clients     map[uuid.UUID]map[*Client]struct{} // by user
	subscribers map[uuid.UUID]map[*Client]struct{} // presence target -> interested clients

	// sender persists messages received over the socket; authorizeChat
	// guards typing/read broadcasts to chats the actor may access;
	// onRead persists a read receipt (best-effort, may be nil).
	sender        Sender
	authorizeChat ChatAuthorizer
	calls         CallSignaler
	onRead        func(ctx context.Context, userID uuid.UUID, chatType string, chatID, messageID uuid.UUID)
	// onOffline records a user's last-seen time when their final connection
	// closes (best-effort; may be nil).
	onOffline func(ctx context.Context, userID uuid.UUID)

	// checkSession confirms a socket's session is still live (nil: no
	// re-check); recheck is how often each socket does so.
	checkSession SessionChecker
	recheck      time.Duration
}

// kickEnvelope selects the sockets to end: one session of a user, or every
// session of a user except Keep (uuid.Nil keeps none).
type kickEnvelope struct {
	UserID    uuid.UUID `json:"u"`
	SessionID uuid.UUID `json:"s"`
	Keep      uuid.UUID `json:"k"`
}

type fanoutEnvelope struct {
	Recipients []uuid.UUID     `json:"r"`
	Frame      json.RawMessage `json:"f"`
}

type presenceEnvelope struct {
	UserID uuid.UUID `json:"u"`
	Online bool      `json:"o"`
}

func NewHub(log *slog.Logger, rdb *redis.Client, resolve RecipientResolver) *Hub {
	return &Hub{
		log:         log,
		rdb:         rdb,
		resolve:     resolve,
		clients:     make(map[uuid.UUID]map[*Client]struct{}),
		subscribers: make(map[uuid.UUID]map[*Client]struct{}),
		recheck:     sessionRecheck,
	}
}

// SetSessionChecker wires the session re-check that ends sockets whose
// session was revoked (audit A-03). every <= 0 keeps the default interval.
func (h *Hub) SetSessionChecker(check SessionChecker, every time.Duration) {
	h.checkSession = check
	if every > 0 {
		h.recheck = every
	}
}

func (h *Hub) recheckEvery() time.Duration { return h.recheck }

// sessionLive re-checks a client's session and records when it last passed.
// A failed check (the database is unreachable) keeps the socket: dropping
// every connection on a blip would only stampede the reconnects, and the
// handshake they would retry needs the same database.
func (h *Hub) sessionLive(c *Client) bool {
	if h.checkSession == nil {
		return true
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	live, err := h.checkSession(ctx, c.userID, c.sessionID)
	if err != nil {
		h.log.Warn("ws: session re-check failed", "error", err)
		return true
	}
	if live {
		c.checkedAt.Store(time.Now().UnixNano())
	}
	return live
}

// sessionFresh is sessionLive with a short memory, for the frames a client
// sends: acting on one needs a recent answer, not a query per typing event.
func (h *Hub) sessionFresh(c *Client) bool {
	if time.Since(time.Unix(0, c.checkedAt.Load())) < inboundRecheck {
		return true
	}
	return h.sessionLive(c)
}

// KickSession ends the sockets of one revoked session, on every instance.
func (h *Hub) KickSession(userID, sessionID uuid.UUID) {
	h.publishKick(kickEnvelope{UserID: userID, SessionID: sessionID})
}

// KickUser ends every socket of a user except those of keep (uuid.Nil keeps
// none), on every instance: logout everywhere, a password change, a
// deactivation, a role change.
func (h *Hub) KickUser(userID, keep uuid.UUID) {
	h.publishKick(kickEnvelope{UserID: userID, Keep: keep})
}

func (h *Hub) publishKick(env kickEnvelope) {
	payload, _ := json.Marshal(env)
	if err := h.rdb.Publish(context.Background(), channelKick, payload).Err(); err != nil {
		// Other instances fall back to their re-check; this one need not wait.
		h.log.Warn("ws: kick publish failed", "error", err)
		h.onKick(payload)
	}
}

func (h *Hub) onKick(payload []byte) {
	var env kickEnvelope
	if err := json.Unmarshal(payload, &env); err != nil || env.UserID == uuid.Nil {
		return
	}
	h.mu.RLock()
	var ending []*Client
	for c := range h.clients[env.UserID] {
		switch {
		case env.SessionID != uuid.Nil && c.sessionID != env.SessionID:
		case env.Keep != uuid.Nil && c.sessionID == env.Keep:
		default:
			ending = append(ending, c)
		}
	}
	h.mu.RUnlock()
	for _, c := range ending {
		c.end("session revoked")
	}
}

// SetHandlers wires inbound-message behaviour after construction. onRead
// may be nil to disable read-receipt persistence.
func (h *Hub) SetHandlers(sender Sender, authorizeChat ChatAuthorizer, onRead func(ctx context.Context, userID uuid.UUID, chatType string, chatID, messageID uuid.UUID)) {
	h.sender = sender
	h.authorizeChat = authorizeChat
	h.onRead = onRead
}

// SetPresenceSink wires a callback invoked when a user's last connection
// closes, so their last-seen time can be persisted. May be nil.
func (h *Hub) SetPresenceSink(onOffline func(ctx context.Context, userID uuid.UUID)) {
	h.onOffline = onOffline
}

// SetCallSignaler wires the voice-call signaling handler. May be nil to
// disable calling.
func (h *Hub) SetCallSignaler(cs CallSignaler) { h.calls = cs }

// Run subscribes to the Redis channels and delivers received events to
// local clients until ctx is cancelled. It should run in its own goroutine.
func (h *Hub) Run(ctx context.Context) {
	sub := h.rdb.Subscribe(ctx, channelFanout, channelPresence, channelBroadcast, channelKick)
	defer sub.Close()

	ch := sub.Channel()
	for {
		select {
		case <-ctx.Done():
			return
		case msg, ok := <-ch:
			if !ok {
				return
			}
			switch msg.Channel {
			case channelFanout:
				h.onFanout([]byte(msg.Payload))
			case channelPresence:
				h.onPresence([]byte(msg.Payload))
			case channelBroadcast:
				h.onBroadcast([]byte(msg.Payload))
			case channelKick:
				h.onKick([]byte(msg.Payload))
			}
		}
	}
}

func (h *Hub) addClient(c *Client) {
	h.mu.Lock()
	if h.clients[c.userID] == nil {
		h.clients[c.userID] = make(map[*Client]struct{})
	}
	h.clients[c.userID][c] = struct{}{}
	h.mu.Unlock()

	metrics.WSConnect()
	h.markPresence(c.userID, true)
}

func (h *Hub) removeClient(c *Client) {
	h.mu.Lock()
	set := h.clients[c.userID]
	if _, present := set[c]; !present {
		h.mu.Unlock()
		return // already removed; keep this idempotent
	}
	delete(set, c)
	if len(set) == 0 {
		delete(h.clients, c.userID)
	}
	for target := range c.subs {
		if s := h.subscribers[target]; s != nil {
			delete(s, c)
			if len(s) == 0 {
				delete(h.subscribers, target)
			}
		}
	}
	close(c.send)
	close(c.done)
	h.mu.Unlock()

	metrics.WSDisconnect()
	h.markPresence(c.userID, false)
}

// subscribePresence records the client's interest in the given users and
// immediately reports those already online.
func (h *Hub) subscribePresence(c *Client, targets []uuid.UUID) {
	h.mu.Lock()
	for _, t := range targets {
		c.subs[t] = struct{}{}
		if h.subscribers[t] == nil {
			h.subscribers[t] = make(map[*Client]struct{})
		}
		h.subscribers[t][c] = struct{}{}
	}
	h.mu.Unlock()

	for _, t := range targets {
		if online, _ := h.rdb.Exists(context.Background(), presenceKey(t)).Result(); online == 1 {
			c.enqueue(encode(EventUserOnline, presenceData{UserID: t}))
		}
	}
}

// --- presence ---

// presenceTTL bounds how long a presence counter can outlive the connections
// it counts. Without it an unclean shutdown (crash, redeploy, killed
// container) leaves the key set forever: the user reads as "online" with no
// socket behind them, so call invites are accepted and then delivered to
// nobody — the caller rings until timeout. Every live connection refreshes the
// key from its ping tick (pingPeriod 54s), so a real session never expires
// while a stale one is reaped within this window.
const presenceTTL = 150 * time.Second

func presenceKey(u uuid.UUID) string { return "kisy:presence:" + u.String() }

// touchPresence extends the presence key for a still-connected user. Called
// from each client's ping tick; a no-op if the key is already gone.
func (h *Hub) touchPresence(userID uuid.UUID) {
	h.rdb.Expire(context.Background(), presenceKey(userID), presenceTTL)
}

// markPresence maintains a cross-instance connection counter and publishes
// online/offline transitions on the first/last connection.
func (h *Hub) markPresence(userID uuid.UUID, online bool) {
	ctx := context.Background()
	key := presenceKey(userID)

	if online {
		n, err := h.rdb.Incr(ctx, key).Result()
		if err != nil {
			h.log.Warn("presence incr failed", "error", err)
			return
		}
		// Bound the key's lifetime so it cannot survive the process that owns
		// the connections (see presenceTTL); refreshed by touchPresence.
		h.rdb.Expire(ctx, key, presenceTTL)
		if n == 1 {
			h.publishPresence(userID, true)
		}
		return
	}

	n, err := h.rdb.Decr(ctx, key).Result()
	if err != nil {
		h.log.Warn("presence decr failed", "error", err)
		return
	}
	if n <= 0 {
		h.rdb.Del(ctx, key)
		if h.onOffline != nil {
			h.onOffline(ctx, userID)
		}
		h.publishPresence(userID, false)
	}
}

func (h *Hub) publishPresence(userID uuid.UUID, online bool) {
	payload, _ := json.Marshal(presenceEnvelope{UserID: userID, Online: online})
	if err := h.rdb.Publish(context.Background(), channelPresence, payload).Err(); err != nil {
		h.log.Warn("presence publish failed", "error", err)
	}
}

func (h *Hub) onPresence(payload []byte) {
	var env presenceEnvelope
	if err := json.Unmarshal(payload, &env); err != nil {
		return
	}
	event := EventUserOnline
	if !env.Online {
		event = EventUserOffline
	}
	frame := encode(event, presenceData{UserID: env.UserID})

	h.mu.RLock()
	defer h.mu.RUnlock()
	for c := range h.subscribers[env.UserID] {
		c.enqueue(frame)
	}
}

// --- targeted fan-out ---

// publishToChat resolves the recipients of a chat and fans a frame out to
// them across all instances.
func (h *Hub) publishToChat(chatType string, chatID uuid.UUID, frame []byte) {
	recipients, err := h.resolve(context.Background(), chatType, chatID)
	if err != nil {
		h.log.Warn("recipient resolve failed", "chatType", chatType, "error", err)
		return
	}
	h.publishToUsers(recipients, frame)
}

func (h *Hub) publishToUsers(recipients []uuid.UUID, frame []byte) {
	env := fanoutEnvelope{Recipients: recipients, Frame: frame}
	payload, _ := json.Marshal(env)
	if err := h.rdb.Publish(context.Background(), channelFanout, payload).Err(); err != nil {
		h.log.Warn("fanout publish failed", "error", err)
	}
}

// Broadcast delivers a frame to every connected client on every instance, for
// app-wide events (e.g. the shared rating board changed).
func (h *Hub) broadcast(frame []byte) {
	if err := h.rdb.Publish(context.Background(), channelBroadcast, frame).Err(); err != nil {
		h.log.Warn("broadcast publish failed", "error", err)
	}
}

func (h *Hub) onBroadcast(frame []byte) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	for _, set := range h.clients {
		for c := range set {
			c.enqueue(frame)
		}
	}
}

func (h *Hub) onFanout(payload []byte) {
	var env fanoutEnvelope
	if err := json.Unmarshal(payload, &env); err != nil {
		return
	}
	h.mu.RLock()
	defer h.mu.RUnlock()
	for _, uid := range env.Recipients {
		for c := range h.clients[uid] {
			c.enqueue(env.Frame)
		}
	}
}
