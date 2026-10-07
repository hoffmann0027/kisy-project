package ws

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"testing"

	"github.com/google/uuid"
)

// Audit A-37: a read receipt was broadcast to the chat's members whatever
// message id it named — one from another chat included. The hub now announces
// only a position the store accepted.
func TestARefusedReadReceiptIsNotBroadcast(t *testing.T) {
	// No Redis behind this hub: reaching the broadcast would panic, which is
	// the failure this test is watching for.
	h := NewHub(slog.New(slog.NewTextHandler(io.Discard, nil)), nil, nil)
	stored := 0
	h.SetHandlers(nil, nil, func(context.Context, uuid.UUID, string, uuid.UUID, uuid.UUID) bool {
		stored++
		return false // the message is not in this chat
	})

	c := &Client{hub: h, userID: uuid.New(), subs: map[uuid.UUID]struct{}{}, send: make(chan []byte, 4)}
	data, _ := json.Marshal(map[string]any{"chatType": "private", "chatId": uuid.New(), "messageId": uuid.New()})

	func() {
		defer func() {
			if r := recover(); r != nil {
				t.Fatalf("a refused read receipt went on to be broadcast (%v)", r)
			}
		}()
		h.handleRead(context.Background(), c, data)
	}()
	if stored != 1 {
		t.Fatalf("the read hook was consulted %d times, want 1", stored)
	}
}
