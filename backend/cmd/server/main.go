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

	if cfg.Env != "production" {
		if err := postgres.Migrate(cfg.Postgres.DSN(), "migrations"); err != nil {
			return err
		}
		log.Info("migrations applied")
	}

	pgPool, err := postgres.NewPool(ctx, cfg.Postgres)
	if err != nil {
		return err
	}
	defer pgPool.Close()
	log.Info("connected to postgres")

	redisClient, err := kisyredis.NewClient(ctx, cfg.Redis)
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
			})

			r.Route("/favorites", func(r chi.Router) {
				m.favoritesHandler.Routes(r)
			})

			r.Route("/notifications", func(r chi.Router) {
				m.notificationsHandler.Routes(r)
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
			m.reactionsHandler.Routes(r)
			m.readstateHandler.Routes(r)
		})

		// WebSocket upgrade authenticates from the access cookie or an
		// access_token query parameter, so it sits outside RequireAuth.
		r.Handle("/ws", m.wsHandler)
	})

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
