// Command server is the KISY backend entrypoint. At this stage it only
// establishes the platform foundation: configuration, database and cache
// connectivity, and a health-checked HTTP server. Business modules
// (auth, chats, messages, ...) are wired in subsequent stages.
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
	"github.com/redis/go-redis/v9"

	"kisy-backend/internal/config"
	"kisy-backend/internal/platform/logger"
	"kisy-backend/internal/platform/postgres"
	kisyredis "kisy-backend/internal/platform/redis"
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

	router := newRouter(log, pgPool, redisClient)

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

func newRouter(log *slog.Logger, pg *pgxpool.Pool, rdb *redis.Client) http.Handler {
	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)
	r.Use(requestLogger(log))
	r.Use(middleware.Timeout(30 * time.Second))

	// Liveness: process is up, no dependency checks.
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		httpresponse.OK(w, r, http.StatusOK, map[string]string{"status": "alive"})
	})

	// Readiness: process can actually serve traffic.
	r.Get("/ready", func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()

		if err := pg.Ping(ctx); err != nil {
			httpresponse.Fail(w, r, http.StatusServiceUnavailable, httpresponse.ErrInternal, "database unavailable")
			return
		}
		if err := rdb.Ping(ctx).Err(); err != nil {
			httpresponse.Fail(w, r, http.StatusServiceUnavailable, httpresponse.ErrInternal, "cache unavailable")
			return
		}

		httpresponse.OK(w, r, http.StatusOK, map[string]string{"status": "ready"})
	})

	r.Route("/api/v1", func(r chi.Router) {
		// Business endpoints are added in subsequent stages.
	})

	return r
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
