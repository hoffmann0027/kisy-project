package main

import (
	"context"

	"kisy-backend/internal/moderation"
	"kisy-backend/internal/notifications"
)

// moderationNotifier delivers moderation notices through the notifications
// module: stored for each recipient, a live event, and a push.
type moderationNotifier struct {
	notifications *notifications.Service
}

func (n moderationNotifier) Notify(ctx context.Context, notice moderation.Notice) error {
	return n.notifications.Announce(ctx, notice.Recipients, notifications.Announcement{
		Type:      notifications.TypeGroupSanction,
		Payload:   notice.Payload,
		PushTitle: "KISY",
		PushBody:  notice.Text,
		URL:       notice.URL,
	})
}
