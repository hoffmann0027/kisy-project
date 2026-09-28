package main

import (
	"context"

	"github.com/google/uuid"

	"kisy-backend/internal/blocks"
	"kisy-backend/internal/calls"
	"kisy-backend/internal/chats"
	"kisy-backend/internal/messages"
)

// Blocking (audit E-02) is decided in one place and applied in four: starting
// a conversation, writing in one, ringing someone, and — in SQL, because it
// has to filter rows rather than refuse a request — the feed, the walls and
// people search.
//
// A block is one-sided and silent, but it works in both directions here: if
// either side blocked the other, neither may write or call. Anything else
// would let the blocked person keep the conversation going.
func wireBlocks(svc *blocks.Service, chatsSvc *chats.Service, msgs *messages.Service, callsSvc *calls.Service) {
	between := func(ctx context.Context, a, b uuid.UUID) (bool, error) {
		return svc.Between(ctx, a, b)
	}
	chatsSvc.SetBlockCheck(between)
	callsSvc.SetBlockCheck(between)

	// A message names its chat, not its recipient, so the other side is looked
	// up first. A chat that no longer exists blocks nothing: the send fails on
	// its own authorization.
	msgs.SetBlockCheck(func(ctx context.Context, chatID, senderID uuid.UUID) (bool, error) {
		ids, err := chatsSvc.ParticipantIDs(ctx, chatID)
		if err != nil || len(ids) != 2 {
			return false, err
		}
		other := ids[0]
		if other == senderID {
			other = ids[1]
		}
		return svc.Between(ctx, senderID, other)
	})
}
