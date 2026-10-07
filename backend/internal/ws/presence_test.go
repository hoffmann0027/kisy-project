package ws

import (
	"context"
	"io"
	"log/slog"
	"testing"

	"github.com/google/uuid"
)

// Audit A-19: presence.subscribe accepted any user id, so anyone could follow
// when anyone came online. The hub now asks its filter first and registers
// nothing it refuses.
func TestPresenceSubscriptionRegistersOnlyWhatTheFilterAllows(t *testing.T) {
	h := NewHub(slog.New(slog.NewTextHandler(io.Discard, nil)), nil, nil)
	stranger := uuid.New()
	var asked []uuid.UUID
	h.SetPresenceFilter(func(_ context.Context, _ uuid.UUID, targets []uuid.UUID) ([]uuid.UUID, error) {
		asked = targets
		return nil, nil // nobody here is a chat partner
	})

	c := &Client{hub: h, userID: uuid.New(), subs: map[uuid.UUID]struct{}{}, send: make(chan []byte, 4)}
	h.subscribePresence(c, []uuid.UUID{stranger})

	if len(asked) != 1 || asked[0] != stranger {
		t.Fatalf("the filter was not consulted: %v", asked)
	}
	if _, watching := h.subscribers[stranger]; watching {
		t.Fatal("a stranger's presence was subscribed to")
	}
	if len(c.subs) != 0 {
		t.Fatalf("client subscriptions: %v", c.subs)
	}

	// A list longer than any chat list is cut before it reaches the database.
	many := make([]uuid.UUID, maxPresenceTargets+50)
	for i := range many {
		many[i] = uuid.New()
	}
	h.subscribePresence(c, many)
	if len(asked) != maxPresenceTargets {
		t.Fatalf("filter got %d targets, want at most %d", len(asked), maxPresenceTargets)
	}
}
