package main

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	goredis "github.com/redis/go-redis/v9"

	"kisy-backend/internal/access"
	"kisy-backend/internal/admin"
	"kisy-backend/internal/attachments"
	"kisy-backend/internal/audit"
	"kisy-backend/internal/auth"
	"kisy-backend/internal/auth/token"
	"kisy-backend/internal/avatars"
	"kisy-backend/internal/boards"
	"kisy-backend/internal/bootstrap"
	"kisy-backend/internal/calls"
	"kisy-backend/internal/chatmedia"
	"kisy-backend/internal/chats"
	"kisy-backend/internal/conditions"
	"kisy-backend/internal/config"
	"kisy-backend/internal/e2ee"
	"kisy-backend/internal/favorites"
	"kisy-backend/internal/feedback"
	"kisy-backend/internal/groups"
	"kisy-backend/internal/invitations"
	"kisy-backend/internal/messages"
	"kisy-backend/internal/notes"
	"kisy-backend/internal/notifications"
	"kisy-backend/internal/platform/ratelimit"
	"kisy-backend/internal/push"
	"kisy-backend/internal/rating"
	"kisy-backend/internal/reactions"
	"kisy-backend/internal/readstate"
	"kisy-backend/internal/search"
	"kisy-backend/internal/users"
	"kisy-backend/internal/voting"
	"kisy-backend/internal/ws"
)

// modules holds every wired HTTP handler and background component the
// router needs. buildModules is the composition root: it is the one place
// that knows how the layers fit together.
type modules struct {
	authHandler          *auth.Handler
	authMW               *auth.Middleware
	usersHandler         *users.Handler
	invitesHandler       *invitations.Handler
	chatsHandler         *chats.Handler
	chatmediaHandler     *chatmedia.Handler
	groupsHandler        *groups.Handler
	avatarsHandler       *avatars.Handler
	messagesHandler      *messages.Handler
	attachmentsHandler   *attachments.Handler
	reactionsHandler     *reactions.Handler
	readstateHandler     *readstate.Handler
	favoritesHandler     *favorites.Handler
	feedbackHandler      *feedback.Handler
	notesHandler         *notes.Handler
	conditionsHandler    *conditions.Handler
	searchHandler        *search.Handler
	pushHandler          *push.Handler
	notificationsHandler *notifications.Handler
	boardsHandler        *boards.Handler
	ratingHandler        *rating.Handler
	votingHandler        *voting.Handler
	callsHandler         *calls.Handler
	e2eeHandler          *e2ee.Handler
	adminHandler         *admin.Handler
	wsHandler            *ws.Handler
	hub                  *ws.Hub
	limiter              *ratelimit.Limiter
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

	// Avatar storage (used by both the users and groups handlers).
	avatarsSvc := avatars.NewService(pool, avatars.NewPostgresRepository())
	avatarsHandler := avatars.NewHandler(avatarsSvc)

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
	usersHandler := users.NewHandler(usersSvc, avatarsSvc,
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

	// --- users service (needed by chats enrichment below) ---
	usersProfile := func(ctx context.Context, id uuid.UUID) (any, bool) {
		u, err := usersRepo.GetByID(ctx, pool, id)
		if err != nil {
			return nil, false
		}
		return u.ToDTO(), true
	}

	// --- chats ---
	chatsSvc := chats.NewService(pool, chatsRepo, userLevel)
	chatsSvc.SetProfileLoader(chats.ProfileLoader(usersProfile))
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
	groupsSvc.SetProfileLoader(groups.ProfileLoader(usersProfile))
	groupsHandler := groups.NewHandler(groupsSvc, avatarsSvc,
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

	// --- chat media aggregation (context panel tabs: media/files/links) ---
	chatmediaSvc := chatmedia.NewService(pool, chatmedia.NewPostgresRepository(), chatmedia.Authorizer{
		Private: func(ctx context.Context, chatID, actorID uuid.UUID) error {
			ok, err := chatsSvc.IsParticipant(ctx, chatID, actorID)
			if err != nil {
				return err
			}
			if !ok {
				return chatmedia.ErrNotFound
			}
			return nil
		},
		Group: func(ctx context.Context, groupID, actorID uuid.UUID, actorLevel int) error {
			err := groupsSvc.EnsureMember(ctx, groupID, groups.ActorMeta{UserID: actorID, RoleLevel: actorLevel})
			switch {
			case errors.Is(err, groups.ErrNotFound):
				return chatmedia.ErrNotFound
			case errors.Is(err, groups.ErrNotMember):
				return chatmedia.ErrForbidden
			default:
				return err
			}
		},
	})
	chatmediaHandler := chatmedia.NewHandler(chatmediaSvc, func(r *http.Request) (chatmedia.Actor, bool) {
		claims, ok := auth.ClaimsFromContext(r.Context())
		if !ok {
			return chatmedia.Actor{}, false
		}
		return chatmedia.Actor{UserID: claims.UserID, RoleLevel: claims.RoleLevel}, true
	})

	// --- attachments (files/images stored in the DB) ---
	attachmentsSvc := attachments.NewService(pool, attachments.NewPostgresRepository(), attachments.Limits{
		MaxBytesLeadership: cfg.Upload.MaxBytesLeadership,
		MaxBytesStaff:      cfg.Upload.MaxBytesStaff,
		LeadershipMaxLevel: config.LeadershipMaxLevel,
		ChunkBytes:         cfg.Upload.ChunkBytes,
		SessionTTL:         cfg.Upload.SessionTTL,
	})
	// Reap abandoned chunked-upload sessions hourly (chunks cascade).
	attachmentsSvc.StartSessionCleanup(ctx, time.Hour, log)
	attachmentsSvc.SetMessageAccess(func(ctx context.Context, messageID, actorID uuid.UUID, actorLevel int) bool {
		_, _, err := messagesSvc.ResolveAccessible(ctx, messageID, messages.ActorMeta{UserID: actorID, RoleLevel: actorLevel})
		return err == nil
	})
	messagesSvc.SetAttachments(
		func(ctx context.Context, ids []uuid.UUID, messageID, uploader uuid.UUID) error {
			return attachmentsSvc.Link(ctx, pool, ids, messageID, uploader)
		},
		func(ctx context.Context, messageIDs []uuid.UUID) (map[uuid.UUID][]any, error) {
			byMsg, err := attachmentsSvc.ForMessages(ctx, messageIDs)
			if err != nil {
				return nil, err
			}
			out := make(map[uuid.UUID][]any, len(byMsg))
			for k, v := range byMsg {
				arr := make([]any, len(v))
				for i := range v {
					arr[i] = v[i]
				}
				out[k] = arr
			}
			return out, nil
		},
	)
	// Forwarding (stage D): clearance breadth per chat + sender-name snapshot
	// + attachment copier. Breadth = weakest level that can access the chat:
	// group min_role_level; private chat = the weaker of its participants.
	messagesSvc.SetForwarding(
		messages.ClearanceResolver{
			Private: func(ctx context.Context, chatID uuid.UUID) (int, error) {
				ids, err := chatsSvc.ParticipantIDs(ctx, chatID)
				if err != nil {
					return 0, err
				}
				breadth := access.CEOLevel
				for _, id := range ids {
					if lvl, ok := userLevel(ctx, id); ok && lvl > breadth {
						breadth = lvl
					}
				}
				return breadth, nil
			},
			Group: func(ctx context.Context, groupID uuid.UUID) (int, error) {
				return groupsSvc.ClearanceLevel(ctx, groupID)
			},
		},
		func(ctx context.Context, userID uuid.UUID) (string, bool) {
			u, err := usersRepo.GetByID(ctx, pool, userID)
			if err != nil {
				return "", false
			}
			return u.DisplayName, true
		},
		func(ctx context.Context, sourceMessageID, newMessageID, uploader uuid.UUID) ([]any, error) {
			dtos, err := attachmentsSvc.CopyToMessage(ctx, sourceMessageID, newMessageID, uploader)
			if err != nil {
				return nil, err
			}
			out := make([]any, len(dtos))
			for i := range dtos {
				out[i] = dtos[i]
			}
			return out, nil
		},
	)

	attachmentsHandler := attachments.NewHandler(attachmentsSvc, func(r *http.Request) (attachments.Actor, bool) {
		claims, ok := auth.ClaimsFromContext(r.Context())
		if !ok {
			return attachments.Actor{}, false
		}
		return attachments.Actor{UserID: claims.UserID, RoleLevel: claims.RoleLevel}, true
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
	wsPublisher := ws.NewPublisher(hub)
	limiter := ratelimit.NewLimiter(rdb, log)
	messagesSvc.SetPublisher(wsPublisher)
	// Real-time profile/group propagation (Stage B: name/avatar changes).
	usersSvc.SetBroadcaster(func(_ context.Context, audience []uuid.UUID, profile users.DTO) {
		wsPublisher.PublishUserUpdated(audience, profile)
	})
	groupsSvc.SetChangePublisher(wsPublisher.PublishGroupChanged)

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

	// --- reactions ---
	reactionsSvc := reactions.NewService(pool, reactions.NewPostgresRepository(), messagesSvc)
	reactionsSvc.SetPublisher(wsPublisher)
	messagesSvc.SetReactionLoader(reactionsSvc.Loader)
	reactionsHandler := reactions.NewHandler(reactionsSvc, func(r *http.Request) (reactions.Actor, bool) {
		claims, ok := auth.ClaimsFromContext(r.Context())
		if !ok {
			return reactions.Actor{}, false
		}
		return reactions.Actor{UserID: claims.UserID, RoleLevel: claims.RoleLevel}, true
	})

	// --- read state / unread counters ---
	readstateSvc := readstate.NewService(pool, readstate.NewPostgresRepository(), readstate.ChatAuthorizer(chatAuthorizer))
	chatsSvc.SetUnreadLoader(readstateSvc.UnreadForPrivateChats)
	chatsSvc.SetOtherReadLoader(readstateSvc.OthersLastReadPrivate)
	// Per-message "read by N of M" for group chats.
	messagesSvc.SetGroupReadLoader(func(ctx context.Context, chatID uuid.UUID) (map[uuid.UUID]time.Time, int, error) {
		reads, err := readstateSvc.GroupReads(ctx, chatID)
		if err != nil {
			return nil, 0, err
		}
		members, err := groupsSvc.MemberIDs(ctx, chatID)
		if err != nil {
			return nil, 0, err
		}
		return reads, len(members), nil
	})
	readstateHandler := readstate.NewHandler(readstateSvc, func(r *http.Request) (readstate.Actor, bool) {
		claims, ok := auth.ClaimsFromContext(r.Context())
		if !ok {
			return readstate.Actor{}, false
		}
		return readstate.Actor{UserID: claims.UserID, RoleLevel: claims.RoleLevel}, true
	})

	// --- favorites / pinned chats ---
	favoritesSvc := favorites.NewService(pool, favorites.NewPostgresRepository(), favorites.ChatAuthorizer(chatAuthorizer))
	favoritesHandler := favorites.NewHandler(favoritesSvc, func(r *http.Request) (favorites.Actor, bool) {
		claims, ok := auth.ClaimsFromContext(r.Context())
		if !ok {
			return favorites.Actor{}, false
		}
		return favorites.Actor{UserID: claims.UserID, RoleLevel: claims.RoleLevel}, true
	})

	// --- full-text message search ---
	searchSvc := search.NewService(pool, log)
	messagesSvc.SetIndexer(searchSvc)
	searchHandler := search.NewHandler(searchSvc, func(r *http.Request) (uuid.UUID, bool) {
		claims, ok := auth.ClaimsFromContext(r.Context())
		if !ok {
			return uuid.Nil, false
		}
		return claims.UserID, true
	})

	// --- feedback & suggestions ---
	feedbackSvc := feedback.NewService(pool, feedback.NewPostgresRepository())
	feedbackHandler := feedback.NewHandler(feedbackSvc, func(r *http.Request) (feedback.Actor, bool) {
		claims, ok := auth.ClaimsFromContext(r.Context())
		if !ok {
			return feedback.Actor{}, false
		}
		return feedback.Actor{UserID: claims.UserID, RoleLevel: claims.RoleLevel}, true
	})

	// --- personal notes ---
	notesSvc := notes.NewService(pool, notes.NewPostgresRepository())
	notesHandler := notes.NewHandler(notesSvc, func(r *http.Request) (uuid.UUID, bool) {
		claims, ok := auth.ClaimsFromContext(r.Context())
		if !ok {
			return uuid.Nil, false
		}
		return claims.UserID, true
	})

	// --- promotion conditions (level-up ladder) ---
	conditionsSvc := conditions.NewService(pool, conditions.NewPostgresRepository())
	conditionsHandler := conditions.NewHandler(conditionsSvc, func(r *http.Request) (conditions.Actor, bool) {
		claims, ok := auth.ClaimsFromContext(r.Context())
		if !ok {
			return conditions.Actor{}, false
		}
		return conditions.Actor{UserID: claims.UserID, RoleLevel: claims.RoleLevel}, true
	})

	// --- notifications (@mentions) ---
	usernameResolver := func(ctx context.Context, username string) (uuid.UUID, bool) {
		u, err := usersRepo.GetByUsername(ctx, pool, username)
		if err != nil || !u.IsActive {
			return uuid.Nil, false
		}
		return u.ID, true
	}
	notificationsRepo := notifications.NewPostgresRepository()
	notificationsSvc := notifications.NewService(pool, notificationsRepo, recipientResolver, usernameResolver, notificationsRepo, wsPublisher)
	messagesSvc.SetNotifier(notificationsSvc)

	// --- web push ---
	pushSvc := push.NewService(pool, push.NewPostgresRepository(), log, cfg.VAPIDPublicKey, cfg.VAPIDPrivateKey, cfg.VAPIDSubject)
	notificationsSvc.SetPusher(pushSvc)
	pushHandler := push.NewHandler(pushSvc, func(r *http.Request) (uuid.UUID, bool) {
		claims, ok := auth.ClaimsFromContext(r.Context())
		if !ok {
			return uuid.Nil, false
		}
		return claims.UserID, true
	})
	notificationsHandler := notifications.NewHandler(notificationsSvc, func(r *http.Request) (uuid.UUID, bool) {
		claims, ok := auth.ClaimsFromContext(r.Context())
		if !ok {
			return uuid.Nil, false
		}
		return claims.UserID, true
	})

	hub.SetHandlers(messagesSvc, chatAuthorizer, readstateSvc.PersistRead)
	// SetPresenceSink is wired below once the calls service exists, so a user's
	// last-connection close both records last-seen and ends any in-flight call.

	// --- task boards (per group) ---
	boardsSvc := boards.NewService(pool, boards.NewPostgresRepository(), boards.Access{
		EnsureActorMember: func(ctx context.Context, groupID, actorID uuid.UUID, actorLevel int) error {
			return groupsSvc.EnsureMember(ctx, groupID, groups.ActorMeta{UserID: actorID, RoleLevel: actorLevel})
		},
		IsFounder: groupsSvc.IsFounder,
		IsMember:  groupsSvc.IsMember,
	})
	boardsSvc.SetPublisher(wsPublisher)
	boardsHandler := boards.NewHandler(boardsSvc, func(r *http.Request) (boards.Actor, bool) {
		claims, ok := auth.ClaimsFromContext(r.Context())
		if !ok {
			return boards.Actor{}, false
		}
		return boards.Actor{UserID: claims.UserID, RoleLevel: claims.RoleLevel}, true
	})

	// --- rating board (projects → in progress → done + profit ledger) ---
	ratingSvc := rating.NewService(pool, rating.NewPostgresRepository())
	ratingSvc.SetChangePublisher(wsPublisher.PublishRatingChanged)
	ratingHandler := rating.NewHandler(ratingSvc, func(r *http.Request) (rating.Actor, bool) {
		claims, ok := auth.ClaimsFromContext(r.Context())
		if !ok {
			return rating.Actor{}, false
		}
		return rating.Actor{UserID: claims.UserID, RoleLevel: claims.RoleLevel}, true
	})

	// --- voting (CEO-created polls) ---
	votingSvc := voting.NewService(pool, voting.NewPostgresRepository(), auditRec)
	votingSvc.SetChangePublisher(wsPublisher.PublishPollChanged)
	votingHandler := voting.NewHandler(votingSvc, func(r *http.Request) (voting.Actor, bool) {
		claims, ok := auth.ClaimsFromContext(r.Context())
		if !ok {
			return voting.Actor{}, false
		}
		return voting.Actor{UserID: claims.UserID, RoleLevel: claims.RoleLevel}, true
	})

	// --- voice calls (WebRTC signaling over WS + call journal) ---
	callsSvc := calls.NewService(
		pool,
		calls.NewPostgresRepository(),
		calls.NewRedisStore(rdb),
		chatsSvc, // ChatAccess: IsParticipant / ParticipantIDs
		auditRec,
		calls.ICESettings{
			STUNURLs:   cfg.ICE.STUNURLs,
			TURNURLs:   cfg.ICE.TURNURLs,
			TURNSecret: cfg.ICE.TURNSecret,
			TURNTTL:    cfg.ICE.TURNTTL,
		},
		log,
	)
	callsSvc.SetPublisher(wsPublisher)
	callsSvc.SetProfileLookup(func(ctx context.Context, id uuid.UUID) (string, *string, bool) {
		u, err := usersRepo.GetByID(ctx, pool, id)
		if err != nil || !u.IsActive {
			return "", nil, false
		}
		return u.DisplayName, u.AvatarURL, true
	})
	callsSvc.SetRateGuard(func(ctx context.Context, id uuid.UUID) bool {
		return limiter.Allow(ctx, "call.invite", id.String(), 10, time.Minute)
	})
	hub.SetCallSignaler(callSignalAdapter{svc: callsSvc})
	// A user's final disconnect records last-seen and tears down any call they
	// were on (prevents the other party ringing forever / stale busy markers).
	hub.SetPresenceSink(func(ctx context.Context, userID uuid.UUID) {
		usersSvc.TouchLastSeen(ctx, userID)
		callsSvc.HandleDisconnect(ctx, userID)
	})
	callsHandler := calls.NewHandler(callsSvc, func(r *http.Request) (calls.Actor, bool) {
		claims, ok := auth.ClaimsFromContext(r.Context())
		if !ok {
			return calls.Actor{}, false
		}
		return calls.Actor{UserID: claims.UserID, SessionID: claims.SessionID, RoleLevel: claims.RoleLevel}, true
	})

	// --- E2EE directory & handshake mailbox (docs/e2ee-design.md §5) ---
	e2eeSvc := e2ee.NewService(pool, e2ee.NewPostgresRepository(), e2ee.Authorizer{
		Private: func(ctx context.Context, chatID, actorID uuid.UUID) error {
			ok, err := chatsSvc.IsParticipant(ctx, chatID, actorID)
			if err != nil {
				return err
			}
			if !ok {
				return e2ee.ErrNotFound
			}
			return nil
		},
		Group: func(ctx context.Context, groupID, actorID uuid.UUID, actorLevel int) error {
			err := groupsSvc.EnsureMember(ctx, groupID, groups.ActorMeta{UserID: actorID, RoleLevel: actorLevel})
			switch {
			case errors.Is(err, groups.ErrNotFound):
				return e2ee.ErrNotFound
			case errors.Is(err, groups.ErrNotMember):
				return e2ee.ErrForbidden
			default:
				return err
			}
		},
	})
	e2eeSvc.SetPublisher(wsPublisher)
	e2eeHandler := e2ee.NewHandler(e2eeSvc, func(r *http.Request) (e2ee.Actor, bool) {
		claims, ok := auth.ClaimsFromContext(r.Context())
		if !ok {
			return e2ee.Actor{}, false
		}
		return e2ee.Actor{UserID: claims.UserID, RoleLevel: claims.RoleLevel}, true
	})

	// --- admin (CEO) ---
	adminSvc := admin.NewService(pool, usersRepo, sessionsRepo, auditRec)
	adminHandler := admin.NewHandler(adminSvc, audit.NewReader(pool), func(r *http.Request) (admin.ActorMeta, bool) {
		claims, ok := auth.ClaimsFromContext(r.Context())
		if !ok {
			return admin.ActorMeta{}, false
		}
		m := authHandler.ClientMeta(r)
		return admin.ActorMeta{UserID: claims.UserID, SessionID: claims.SessionID, IPHash: m.IPHash, RequestID: m.RequestID}, true
	})

	wsHandler := ws.NewHandler(hub, func(r *http.Request) (ws.Authenticated, bool) {
		claims := authMW.Authenticate(r)
		if claims == nil {
			return ws.Authenticated{}, false
		}
		return ws.Authenticated{UserID: claims.UserID, SessionID: claims.SessionID, RoleLevel: claims.RoleLevel}, true
	}, cfg.WSAllowedOrigin)

	return &modules{
		authHandler:          authHandler,
		authMW:               authMW,
		usersHandler:         usersHandler,
		invitesHandler:       invitesHandler,
		chatsHandler:         chatsHandler,
		chatmediaHandler:     chatmediaHandler,
		groupsHandler:        groupsHandler,
		avatarsHandler:       avatarsHandler,
		messagesHandler:      messagesHandler,
		attachmentsHandler:   attachmentsHandler,
		reactionsHandler:     reactionsHandler,
		readstateHandler:     readstateHandler,
		favoritesHandler:     favoritesHandler,
		feedbackHandler:      feedbackHandler,
		notesHandler:         notesHandler,
		conditionsHandler:    conditionsHandler,
		searchHandler:        searchHandler,
		pushHandler:          pushHandler,
		notificationsHandler: notificationsHandler,
		boardsHandler:        boardsHandler,
		ratingHandler:        ratingHandler,
		votingHandler:        votingHandler,
		callsHandler:         callsHandler,
		e2eeHandler:          e2eeHandler,
		adminHandler:         adminHandler,
		wsHandler:            wsHandler,
		hub:                  hub,
		limiter:              limiter,
	}, nil
}

// callSignalAdapter converts the ws-layer CallActor into the calls-layer Actor,
// letting *calls.Service satisfy ws.CallSignaler without either package
// importing the other (both are wired here in the composition root).
type callSignalAdapter struct{ svc *calls.Service }

func (a callSignalAdapter) HandleSignal(ctx context.Context, actor ws.CallActor, msgType string, data json.RawMessage) error {
	return a.svc.HandleSignal(ctx, calls.Actor{
		UserID:    actor.UserID,
		SessionID: actor.SessionID,
		RoleLevel: actor.RoleLevel,
	}, msgType, data)
}
