package main

import (
	"context"
	"net/http"
	"regexp"
	"time"

	"github.com/google/uuid"

	"kisy-backend/internal/auth"
	"kisy-backend/internal/chats"
	"kisy-backend/internal/config"
	"kisy-backend/internal/messages"
	"kisy-backend/internal/platform/ratelimit"
)

// accountPolicy turns RATE_* into the limiter's per-scope budgets.
func accountPolicy(c config.AccountRateConfig) ratelimit.AccountPolicy {
	perMinute := func(basic, invited int) ratelimit.Tiered {
		return ratelimit.Tiered{Basic: ratelimit.Rule{Max: basic, Window: time.Minute}, Invited: ratelimit.Rule{Max: invited, Window: time.Minute}}
	}
	perHour := func(basic, invited int) ratelimit.Tiered {
		return ratelimit.Tiered{Basic: ratelimit.Rule{Max: basic, Window: time.Hour}, Invited: ratelimit.Rule{Max: invited, Window: time.Hour}}
	}
	return ratelimit.AccountPolicy{
		ratelimit.ScopeMessages:    perMinute(c.MessagesPerMinuteBasic, c.MessagesPerMinuteInvited),
		ratelimit.ScopeNewChats:    perHour(c.NewChatsPerHourBasic, c.NewChatsPerHourInvited),
		ratelimit.ScopeSearch:      perMinute(c.SearchesPerMinuteBasic, c.SearchesPerMinuteInvited),
		ratelimit.ScopeLinkPreview: perMinute(c.LinkPreviewsPerMinuteBasic, c.LinkPreviewsPerMinuteInvited),
		ratelimit.ScopeUploads:     perHour(c.UploadsPerHourBasic, c.UploadsPerHourInvited),
	}
}

// wireAccountLimits puts the message and new-chat budgets into the services,
// where the REST and WebSocket paths meet.
func wireAccountLimits(a *ratelimit.Accounts, msgs *messages.Service, chatsSvc *chats.Service) {
	msgs.SetSendGate(func(ctx context.Context, actor messages.ActorMeta) error {
		return a.Check(ctx, ratelimit.ScopeMessages, actor.UserID, actor.RoleLevel)
	})
	chatsSvc.SetNewChatGate(func(ctx context.Context, actor chats.ActorMeta) error {
		return a.Check(ctx, ratelimit.ScopeNewChats, actor.UserID, actor.RoleLevel)
	})
}

// Messages and new private chats are limited inside their services (so the
// WebSocket path shares the budget, and reopening an existing chat is free).
// The rest are plain request counts, matched here.
var accountLimitedRoutes = []struct {
	method string
	path   *regexp.Regexp
	scope  string
}{
	{http.MethodGet, regexp.MustCompile(`^/api/v1/search/?$`), ratelimit.ScopeSearch},
	{http.MethodPost, regexp.MustCompile(`^/api/v1/link-preview$`), ratelimit.ScopeLinkPreview},
	// A group or community is a new chat too.
	{http.MethodPost, regexp.MustCompile(`^/api/v1/groups/?$`), ratelimit.ScopeNewChats},
	// Uploads count once per file: the one-shot upload or the start of a
	// chunked one — never each chunk.
	{http.MethodPost, regexp.MustCompile(`^/api/v1/attachments$`), ratelimit.ScopeUploads},
	{http.MethodPost, regexp.MustCompile(`^/api/v1/attachments/init$`), ratelimit.ScopeUploads},
	{http.MethodPost, regexp.MustCompile(`^/api/v1/notes/file$`), ratelimit.ScopeUploads},
	{http.MethodPost, regexp.MustCompile(`^/api/v1/posts/[^/]+/media$`), ratelimit.ScopeUploads},
	{http.MethodPost, regexp.MustCompile(`^/api/v1/users/me/avatar$`), ratelimit.ScopeUploads},
	{http.MethodPost, regexp.MustCompile(`^/api/v1/groups/[^/]+/avatar$`), ratelimit.ScopeUploads},
}

func accountLimitedScope(r *http.Request) (string, bool) {
	for _, route := range accountLimitedRoutes {
		if r.Method == route.method && route.path.MatchString(r.URL.Path) {
			return route.scope, true
		}
	}
	return "", false
}

func claimsIdentity(r *http.Request) (uuid.UUID, int, bool) {
	claims, ok := auth.ClaimsFromContext(r.Context())
	if !ok {
		return uuid.Nil, 0, false
	}
	return claims.UserID, claims.RoleLevel, true
}
