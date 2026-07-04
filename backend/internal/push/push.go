// Package push delivers Web Push notifications to users' subscribed browsers.
// It is best-effort: a nil/disabled service (no VAPID keys) simply does
// nothing, and failed endpoints are pruned.
package push

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"

	webpush "github.com/SherClockHolmes/webpush-go"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Subscription is a browser push endpoint with its encryption keys.
type Subscription struct {
	Endpoint string
	P256dh   string
	Auth     string
}

type Repository interface {
	Upsert(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID, s Subscription) error
	Delete(ctx context.Context, pool *pgxpool.Pool, endpoint string) error
	ListForUser(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID) ([]Subscription, error)
}

type PostgresRepository struct{}

func NewPostgresRepository() *PostgresRepository { return &PostgresRepository{} }

func (r *PostgresRepository) Upsert(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID, s Subscription) error {
	_, err := pool.Exec(ctx, `
		INSERT INTO push_subscriptions (user_id, endpoint, p256dh, auth)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (endpoint) DO UPDATE SET user_id = EXCLUDED.user_id, p256dh = EXCLUDED.p256dh, auth = EXCLUDED.auth`,
		userID, s.Endpoint, s.P256dh, s.Auth)
	if err != nil {
		return fmt.Errorf("push: upsert: %w", err)
	}
	return nil
}

func (r *PostgresRepository) Delete(ctx context.Context, pool *pgxpool.Pool, endpoint string) error {
	if _, err := pool.Exec(ctx, `DELETE FROM push_subscriptions WHERE endpoint = $1`, endpoint); err != nil {
		return fmt.Errorf("push: delete: %w", err)
	}
	return nil
}

func (r *PostgresRepository) ListForUser(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID) ([]Subscription, error) {
	rows, err := pool.Query(ctx, `SELECT endpoint, p256dh, auth FROM push_subscriptions WHERE user_id = $1`, userID)
	if err != nil {
		return nil, fmt.Errorf("push: list: %w", err)
	}
	defer rows.Close()
	var out []Subscription
	for rows.Next() {
		var s Subscription
		if err := rows.Scan(&s.Endpoint, &s.P256dh, &s.Auth); err != nil {
			return nil, fmt.Errorf("push: scan: %w", err)
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

// Service sends pushes and manages subscriptions.
type Service struct {
	pool       *pgxpool.Pool
	repo       Repository
	log        *slog.Logger
	publicKey  string
	privateKey string
	subject    string
}

func NewService(pool *pgxpool.Pool, repo Repository, log *slog.Logger, publicKey, privateKey, subject string) *Service {
	return &Service{pool: pool, repo: repo, log: log, publicKey: publicKey, privateKey: privateKey, subject: subject}
}

// Enabled reports whether VAPID keys are configured.
func (s *Service) Enabled() bool { return s.publicKey != "" && s.privateKey != "" }

// PublicKey returns the VAPID public key for client subscription.
func (s *Service) PublicKey() string { return s.publicKey }

func (s *Service) Subscribe(ctx context.Context, userID uuid.UUID, sub Subscription) error {
	return s.repo.Upsert(ctx, s.pool, userID, sub)
}

func (s *Service) Unsubscribe(ctx context.Context, endpoint string) error {
	return s.repo.Delete(ctx, s.pool, endpoint)
}

// payload is the JSON the service worker's push handler expects.
type payload struct {
	Title string `json:"title"`
	Body  string `json:"body"`
	URL   string `json:"url"`
	Tag   string `json:"tag,omitempty"`
}

// Notify pushes a notification to all of a user's browsers. It runs its own
// goroutine-friendly work synchronously; callers typically invoke it in a
// goroutine. Dead endpoints (404/410) are pruned.
func (s *Service) Notify(ctx context.Context, userID uuid.UUID, title, body, url string) {
	if !s.Enabled() {
		return
	}
	subs, err := s.repo.ListForUser(ctx, s.pool, userID)
	if err != nil {
		s.log.Warn("push list failed", "error", err)
		return
	}
	if len(subs) == 0 {
		return
	}
	data, _ := json.Marshal(payload{Title: title, Body: body, URL: url, Tag: "kisy"})

	for _, sub := range subs {
		resp, err := webpush.SendNotification(data, &webpush.Subscription{
			Endpoint: sub.Endpoint,
			Keys:     webpush.Keys{P256dh: sub.P256dh, Auth: sub.Auth},
		}, &webpush.Options{
			Subscriber:      s.subject,
			VAPIDPublicKey:  s.publicKey,
			VAPIDPrivateKey: s.privateKey,
			TTL:             86400,
		})
		if err != nil {
			s.log.Warn("push send failed", "error", err)
			continue
		}
		func() {
			defer resp.Body.Close()
			if resp.StatusCode == http.StatusNotFound || resp.StatusCode == http.StatusGone {
				_ = s.repo.Delete(ctx, s.pool, sub.Endpoint)
			}
		}()
	}
}
