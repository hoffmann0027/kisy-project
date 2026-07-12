// Command server is the KISY backend entrypoint: configuration, database
// and cache connectivity, module wiring and the HTTP server.
package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5/pgxpool"
	goredis "github.com/redis/go-redis/v9"

	"kisy-backend/internal/config"
	"kisy-backend/internal/platform/logger"
	"kisy-backend/internal/platform/metrics"
	"kisy-backend/internal/platform/postgres"
	"kisy-backend/internal/platform/ratelimit"
	kisyredis "kisy-backend/internal/platform/redis"
	"kisy-backend/internal/platform/security"
	"kisy-backend/internal/platform/spa"
	"kisy-backend/pkg/httpresponse"
)

func main() {
	if err := run(); err != nil {
		slog.Error("server exited with error", "error", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	log := logger.New(cfg.LogLevel)
	slog.SetDefault(log)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	if cfg.RunMigrations {
		if err := postgres.Migrate(cfg.PostgresDSN(), "migrations"); err != nil {
			return err
		}
		log.Info("migrations applied")
	}

	pgPool, err := postgres.NewPool(ctx, cfg.PostgresDSN())
	if err != nil {
		return err
	}
	defer pgPool.Close()
	log.Info("connected to postgres")

	redisClient, err := kisyredis.NewClient(ctx, cfg.Redis, cfg.RedisURL)
	if err != nil {
		return err
	}
	defer redisClient.Close()
	log.Info("connected to redis")

	mods, err := buildModules(ctx, cfg, pgPool, redisClient, log)
	if err != nil {
		return err
	}

	// The hub bridges real-time events across instances via Redis pub/sub.
	go mods.hub.Run(ctx)

	router := newRouter(routerDeps{
		log:           log,
		pg:            pgPool,
		rdb:           redisClient,
		mods:          mods,
		allowedOrigin: cfg.WSAllowedOrigin,
		webDir:        cfg.WebDir,
	})

	srv := &http.Server{
		Addr:              ":" + strconv.Itoa(cfg.HTTPPort),
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
	}

	serveErr := make(chan error, 1)
	go func() {
		log.Info("http server listening", "port", cfg.HTTPPort)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serveErr <- err
			return
		}
		serveErr <- nil
	}()

	select {
	case <-ctx.Done():
		log.Info("shutdown signal received")
	case err := <-serveErr:
		if err != nil {
			return err
		}
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	return srv.Shutdown(shutdownCtx)
}

type routerDeps struct {
	log           *slog.Logger
	pg            *pgxpool.Pool
	rdb           *goredis.Client
	mods          *modules
	allowedOrigin string
	webDir        string
}

func newRouter(d routerDeps) http.Handler {
	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)
	r.Use(security.Headers)
	r.Use(metrics.Middleware)
	r.Use(requestLogger(d.log))
	r.Use(middleware.Timeout(30 * time.Second))

	// Prometheus scrape endpoint. Internal only — the edge proxy does not
	// forward /metrics, so it is unreachable from the public internet.
	r.Handle("/metrics", metrics.Handler())

	// Liveness: process is up, no dependency checks.
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		httpresponse.OK(w, r, http.StatusOK, map[string]string{"status": "alive"})
	})

	// Readiness: process can actually serve traffic.
	r.Get("/ready", func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()

		if err := d.pg.Ping(ctx); err != nil {
			httpresponse.Fail(w, r, http.StatusServiceUnavailable, httpresponse.ErrInternal, "database unavailable")
			return
		}
		if err := d.rdb.Ping(ctx).Err(); err != nil {
			httpresponse.Fail(w, r, http.StatusServiceUnavailable, httpresponse.ErrInternal, "cache unavailable")
			return
		}

		httpresponse.OK(w, r, http.StatusOK, map[string]string{"status": "ready"})
	})

	m := d.mods

	r.Route("/api/v1", func(r chi.Router) {
		// CSRF protection on every state-changing API request
		// (defense-in-depth alongside SameSite=Strict cookies).
		r.Use(security.CSRF(d.allowedOrigin))

		r.Route("/auth", func(r chi.Router) {
			// Brute-force protection on unauthenticated entry points.
			r.Use(perRouteLimits(m.limiter))
			m.authHandler.Routes(r)
		})

		// Everything below requires a valid session.
		r.Group(func(r chi.Router) {
			r.Use(m.authMW.RequireAuth)

			r.Route("/users", func(r chi.Router) {
				m.usersHandler.Routes(r)
			})

			r.Route("/invites", func(r chi.Router) {
				r.Use(m.authMW.RequireClearance(1)) // CEO only
				m.invitesHandler.Routes(r)
			})

			r.Route("/chats", func(r chi.Router) {
				m.chatsHandler.Routes(r)
				// Context-panel tabs: shared media/files/links of a chat.
				m.chatmediaHandler.Routes(r)
				// Per-chat mute (stage G).
				m.notifprefsHandler.ChatRoutes(r)
				// Per-chat archive (UPD3 stage H).
				m.chatfoldersHandler.ChatRoutes(r)
				// Per-chat disappearing timer (UPD3 stage J).
				m.disappearHandler.ChatRoutes(r)
			})

			r.Route("/settings", func(r chi.Router) {
				m.notifprefsHandler.SettingsRoutes(r)
				m.chatfoldersHandler.SettingsRoutes(r)
			})

			// Personal chat folders (UPD3 stage H).
			r.Route("/folders", func(r chi.Router) {
				m.chatfoldersHandler.FolderRoutes(r)
			})

			r.Route("/favorites", func(r chi.Router) {
				m.favoritesHandler.Routes(r)
			})

			r.Route("/notifications", func(r chi.Router) {
				m.notificationsHandler.Routes(r)
			})

			r.Route("/feedback", func(r chi.Router) {
				m.feedbackHandler.Routes(r)
			})

			r.Route("/notes", func(r chi.Router) {
				m.notesHandler.Routes(r)
			})

			r.Route("/conditions", func(r chi.Router) {
				m.conditionsHandler.Routes(r)
			})

			r.Route("/rating", func(r chi.Router) {
				// The rating board is the clan board: open to clearance levels
				// 1–9; the weakest level (10) is "not in a clan" and blocked.
				r.Use(m.authMW.RequireClearance(9))
				m.ratingHandler.Routes(r)
			})

			r.Route("/polls", func(r chi.Router) {
				m.votingHandler.Routes(r)
			})

			r.Route("/calls", func(r chi.Router) {
				// ICE-config/history reads; live signaling runs over /ws.
				r.Use(m.limiter.Limit("calls", 60, time.Minute))
				m.callsHandler.Routes(r)
			})

			r.Route("/search", func(r chi.Router) {
				m.searchHandler.Routes(r)
			})

			r.Group(func(r chi.Router) {
				// Server-side outbound fetch: rate-limit to blunt SSRF probing
				// and abuse of the preview fetcher.
				r.Use(m.limiter.Limit("link-preview", 30, time.Minute))
				m.linkPreviewHandler.Routes(r)
			})

			r.Route("/e2ee", func(r chi.Router) {
				// Key directory + MLS handshake mailbox. Uploads are
				// rate-limited: key material churn is low-frequency.
				r.Use(m.limiter.Limit("e2ee", 120, time.Minute))
				m.e2eeHandler.Routes(r)
			})

			r.Route("/push", func(r chi.Router) {
				m.pushHandler.Routes(r)
			})

			r.Route("/admin", func(r chi.Router) {
				r.Use(m.authMW.RequireClearance(1)) // CEO only
				m.adminHandler.Routes(r)
			})

			r.Route("/groups", func(r chi.Router) {
				// Reads are visibility-filtered; creation is open to any
				// user (the service caps the group's clearance at the
				// creator's own level); deletion is CEO-or-founder.
				m.groupsHandler.Routes(r)
				// Task board endpoints under a group.
				m.boardsHandler.GroupRoutes(r)
			})

			// Board structure/card endpoints (access checked per-board via
			// group membership/founder).
			r.Route("/boards", func(r chi.Router) {
				m.boardsHandler.BoardRoutes(r)
			})

			// Message, reaction and read-state endpoints live at the
			// /api/v1 root (POST/GET/DELETE /messages,
			// POST/DELETE /messages/{id}/reactions, POST /read).
			m.messagesHandler.Routes(r)
			// Scheduled sending (UPD3 stage I): /messages/schedule[d].
			m.scheduledHandler.Routes(r)
			m.attachmentsHandler.Routes(r)
			m.reactionsHandler.Routes(r)
			m.readstateHandler.Routes(r)

			// Avatar images (GET /avatars/{type}/{id}); uploads live under
			// /users/me/avatar and /groups/{id}/avatar.
			m.avatarsHandler.Routes(r)
		})
	})

	// WebSocket upgrade lives at the top-level /ws (the path the frontend and
	// the edge proxy use). It authenticates from the access cookie or an
	// access_token query parameter, so it sits outside RequireAuth and CSRF.
	r.Handle("/ws", m.wsHandler)

	// Serve the built SPA on all remaining paths when a web directory is
	// configured (single-service, same-origin deploy). API, /ws, /health,
	// /ready and /metrics are matched first by chi, so this only catches
	// frontend routes and static assets.
	if d.webDir != "" {
		r.Handle("/*", spa.Handler(d.webDir))
	}

	return r
}

// perRouteLimits applies scoped rate limits inside /auth: login and
// register are the brute-force targets, refresh is chattier by design.
func perRouteLimits(l *ratelimit.Limiter) func(http.Handler) http.Handler {
	login := l.Limit("auth-login", 10, time.Minute)
	register := l.Limit("auth-register", 5, time.Minute)
	refresh := l.Limit("auth-refresh", 60, time.Minute)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Path {
			case "/api/v1/auth/login":
				login(next).ServeHTTP(w, r)
			case "/api/v1/auth/register":
				register(next).ServeHTTP(w, r)
			case "/api/v1/auth/refresh":
				refresh(next).ServeHTTP(w, r)
			default:
				next.ServeHTTP(w, r)
			}
		})
	}
}

func requestLogger(log *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)

			next.ServeHTTP(ww, r)

			log.Info("http request",
				"method", r.Method,
				"path", r.URL.Path,
				"status", ww.Status(),
				"duration_ms", time.Since(start).Milliseconds(),
				"request_id", middleware.GetReqID(r.Context()),
			)
		})
	}
}
