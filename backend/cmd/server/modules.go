package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	goredis "github.com/redis/go-redis/v9"

	"kisy-backend/internal/audit"
	"kisy-backend/internal/auth"
	"kisy-backend/internal/auth/token"
	"kisy-backend/internal/bootstrap"
	"kisy-backend/internal/chats"
	"kisy-backend/internal/config"
	"kisy-backend/internal/groups"
	"kisy-backend/internal/invitations"
	"kisy-backend/internal/messages"
	"kisy-backend/internal/platform/ratelimit"
	"kisy-backend/internal/users"
	"kisy-backend/internal/ws"
)

// modules holds every wired HTTP handler and background component the
// router needs. buildModules is the composition root: it is the one place
// that knows how the layers fit together.
type modules struct {
	authHandler     *auth.Handler
	authMW          *auth.Middleware
	usersHandler    *users.Handler
	invitesHandler  *invitations.Handler
	chatsHandler    *chats.Handler
	groupsHandler   *groups.Handler
	messagesHandler *messages.Handler
	wsHandler       *ws.Handler
	hub             *ws.Hub
	limiter         *ratelimit.Limiter
}

func buildModules(ctx context.Context, cfg *config.Config, pool *pgxpool.Pool, rdb *goredis.Client, log *slog.Logger) (*modules, error) {
	auditRec := audit.NewPostgresRecorder(log)

	usersRepo := users.NewPostgresRepository()
	sessionsRepo := auth.NewPostgresSessionRepository()
	invitesRepo := invitations.NewPostgresRepository()
	chatsRepo := chats.NewPostgresRepository()
	groupsRepo := groups.NewPostgresRepository()
	messagesRepo := messages.NewPostgresRepository()

	if err := bootstrap.EnsureCEO(ctx, pool, usersRepo, auditRec, log, cfg.BootstrapCEOUsername, cfg.BootstrapCEOPassword); err != nil {
		return nil, err
	}

	tokens := token.NewManager(cfg.JWTAccessSecret, cfg.JWTAccessTTL)

	authSvc, err := auth.NewService(pool, usersRepo, sessionsRepo, invitesRepo, auditRec, tokens, cfg.JWTRefreshTTL)
	if err != nil {
		return nil, err
	}
	authMW := auth.NewMiddleware(tokens, sessionsRepo, pool)
	authHandler := auth.NewHandler(authSvc, authMW, cfg.IPHashSalt, cfg.Env == "production")

	// userLevel resolves an active user's clearance; inactive or missing
	// users report ok=false so callers cannot enumerate accounts.
	userLevel := func(ctx context.Context, id uuid.UUID) (int, bool) {
		u, err := usersRepo.GetByID(ctx, pool, id)
		if err != nil || !u.IsActive {
			return 0, false
		}
		return u.RoleID, true
	}

	// --- users ---
	usersSvc := users.NewService(pool, usersRepo, auditRec)
	usersHandler := users.NewHandler(usersSvc,
		func(r *http.Request) (users.Identity, bool) {
			claims, ok := auth.ClaimsFromContext(r.Context())
			if !ok {
				return users.Identity{}, false
			}
			return users.Identity{UserID: claims.UserID, SessionID: claims.SessionID, RoleLevel: claims.RoleLevel}, true
		},
		func(r *http.Request) users.ActorMeta {
			claims, _ := auth.ClaimsFromContext(r.Context())
			m := authHandler.ClientMeta(r)
			am := users.ActorMeta{IPHash: m.IPHash, RequestID: m.RequestID}
			if claims != nil {
				am.SessionID = claims.SessionID
			}
			return am
		},
	)

	// --- invitations ---
	invitesSvc := invitations.NewService(pool, invitesRepo, auditRec, cfg.InvitationTokenTTL)
	invitesHandler := invitations.NewHandler(invitesSvc, func(r *http.Request) (invitations.CreatorMeta, bool) {
		claims, ok := auth.ClaimsFromContext(r.Context())
		if !ok {
			return invitations.CreatorMeta{}, false
		}
		m := authHandler.ClientMeta(r)
		return invitations.CreatorMeta{ActorID: claims.UserID, SessionID: claims.SessionID, IPHash: m.IPHash, RequestID: m.RequestID}, true
	})

	// --- chats ---
	chatsSvc := chats.NewService(pool, chatsRepo, userLevel)
	chatsHandler := chats.NewHandler(chatsSvc, func(r *http.Request) (chats.ActorMeta, bool) {
		claims, ok := auth.ClaimsFromContext(r.Context())
		if !ok {
			return chats.ActorMeta{}, false
		}
		m := authHandler.ClientMeta(r)
		return chats.ActorMeta{UserID: claims.UserID, RoleLevel: claims.RoleLevel, SessionID: claims.SessionID, IPHash: m.IPHash, RequestID: m.RequestID}, true
	})

	// --- groups ---
	groupsSvc := groups.NewService(pool, groupsRepo, auditRec)
	groupsHandler := groups.NewHandler(groupsSvc,
		func(r *http.Request) (groups.ActorMeta, bool) {
			claims, ok := auth.ClaimsFromContext(r.Context())
			if !ok {
				return groups.ActorMeta{}, false
			}
			m := authHandler.ClientMeta(r)
			return groups.ActorMeta{UserID: claims.UserID, RoleLevel: claims.RoleLevel, SessionID: claims.SessionID, IPHash: m.IPHash, RequestID: m.RequestID}, true
		},
		groups.UserLookup(userLevel),
	)

	// --- messages ---
	messagesAuthz := messages.Authorizer{
		Private: func(ctx context.Context, chatID, actorID uuid.UUID) error {
			ok, err := chatsSvc.IsParticipant(ctx, chatID, actorID)
			if err != nil {
				return err
			}
			if !ok {
				return messages.ErrNotFound
			}
			return nil
		},
		Group: func(ctx context.Context, groupID, actorID uuid.UUID, actorLevel int) error {
			err := groupsSvc.EnsureMember(ctx, groupID, groups.ActorMeta{UserID: actorID, RoleLevel: actorLevel})
			switch {
			case errors.Is(err, groups.ErrNotFound):
				return messages.ErrNotFound
			case errors.Is(err, groups.ErrNotMember):
				return messages.ErrForbidden
			default:
				return err
			}
		},
	}
	messagesSvc := messages.NewService(pool, messagesRepo, auditRec, messagesAuthz)
	messagesHandler := messages.NewHandler(messagesSvc, func(r *http.Request) (messages.ActorMeta, bool) {
		claims, ok := auth.ClaimsFromContext(r.Context())
		if !ok {
			return messages.ActorMeta{}, false
		}
		m := authHandler.ClientMeta(r)
		return messages.ActorMeta{UserID: claims.UserID, RoleLevel: claims.RoleLevel, SessionID: claims.SessionID, IPHash: m.IPHash, RequestID: m.RequestID}, true
	})

	// --- websocket ---
	recipientResolver := func(ctx context.Context, chatType string, chatID uuid.UUID) ([]uuid.UUID, error) {
		switch chatType {
		case messages.ChatPrivate:
			return chatsSvc.ParticipantIDs(ctx, chatID)
		case messages.ChatGroup:
			return groupsSvc.MemberIDs(ctx, chatID)
		default:
			return nil, nil
		}
	}
	hub := ws.NewHub(log, rdb, recipientResolver)

	messagesSvc.SetPublisher(ws.NewPublisher(hub))

	chatAuthorizer := ws.ChatAuthorizer(func(ctx context.Context, chatType string, chatID, actorID uuid.UUID, actorLevel int) error {
		switch chatType {
		case messages.ChatPrivate:
			ok, err := chatsSvc.IsParticipant(ctx, chatID, actorID)
			if err != nil {
				return err
			}
			if !ok {
				return errors.New("ws: not a participant")
			}
			return nil
		case messages.ChatGroup:
			return groupsSvc.EnsureMember(ctx, chatID, groups.ActorMeta{UserID: actorID, RoleLevel: actorLevel})
		default:
			return errors.New("ws: unknown chat type")
		}
	})
	hub.SetHandlers(messagesSvc, chatAuthorizer)

	wsHandler := ws.NewHandler(hub, func(r *http.Request) (ws.Authenticated, bool) {
		claims := authMW.Authenticate(r)
		if claims == nil {
			return ws.Authenticated{}, false
		}
		return ws.Authenticated{UserID: claims.UserID, SessionID: claims.SessionID, RoleLevel: claims.RoleLevel}, true
	}, cfg.WSAllowedOrigin)

	return &modules{
		authHandler:     authHandler,
		authMW:          authMW,
		usersHandler:    usersHandler,
		invitesHandler:  invitesHandler,
		chatsHandler:    chatsHandler,
		groupsHandler:   groupsHandler,
		messagesHandler: messagesHandler,
		wsHandler:       wsHandler,
		hub:             hub,
		limiter:         ratelimit.NewLimiter(rdb, log),
	}, nil
}
