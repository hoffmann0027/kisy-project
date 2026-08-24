// Package push delivers notifications to the devices a user is not currently
// looking at: subscribed browsers over Web Push, and the packaged mobile app
// over Firebase Cloud Messaging. Both transports are best-effort and
// independently optional — with no VAPID keys and no FCM credentials the
// service simply does nothing — and dead endpoints are pruned as they surface.
package push

import (
	"context"
	"encoding/json"
	"errors"
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

// Device is one installation of the mobile app, addressed by its FCM
// registration token.
type Device struct {
	Token    string
	Platform string
}

type Repository interface {
	Upsert(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID, s Subscription) error
	Delete(ctx context.Context, pool *pgxpool.Pool, endpoint string) error
	ListForUser(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID) ([]Subscription, error)

	UpsertDevice(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID, d Device) error
	DeleteDevice(ctx context.Context, pool *pgxpool.Pool, token string) error
	ListDevicesForUser(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID) ([]Device, error)
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

// UpsertDevice records (or refreshes) a device registration token. Tokens move
// between users when a phone is handed over or a second account signs in, so a
// conflict rebinds the row instead of failing.
func (r *PostgresRepository) UpsertDevice(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID, d Device) error {
	_, err := pool.Exec(ctx, `
		INSERT INTO device_tokens (user_id, token, platform)
		VALUES ($1, $2, $3)
		ON CONFLICT (token) DO UPDATE SET user_id = EXCLUDED.user_id, platform = EXCLUDED.platform, last_seen_at = now()`,
		userID, d.Token, d.Platform)
	if err != nil {
		return fmt.Errorf("push: upsert device: %w", err)
	}
	return nil
}

func (r *PostgresRepository) DeleteDevice(ctx context.Context, pool *pgxpool.Pool, token string) error {
	if _, err := pool.Exec(ctx, `DELETE FROM device_tokens WHERE token = $1`, token); err != nil {
		return fmt.Errorf("push: delete device: %w", err)
	}
	return nil
}

func (r *PostgresRepository) ListDevicesForUser(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID) ([]Device, error) {
	rows, err := pool.Query(ctx, `SELECT token, platform FROM device_tokens WHERE user_id = $1`, userID)
	if err != nil {
		return nil, fmt.Errorf("push: list devices: %w", err)
	}
	defer rows.Close()
	var out []Device
	for rows.Next() {
		var d Device
		if err := rows.Scan(&d.Token, &d.Platform); err != nil {
			return nil, fmt.Errorf("push: scan device: %w", err)
		}
		out = append(out, d)
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

	// fcm is nil unless Firebase credentials were configured.
	fcm *FCM
}

func NewService(pool *pgxpool.Pool, repo Repository, log *slog.Logger, publicKey, privateKey, subject string) *Service {
	return &Service{pool: pool, repo: repo, log: log, publicKey: publicKey, privateKey: privateKey, subject: subject}
}

// SetFCM wires Firebase delivery for the mobile app. Passing nil leaves the
// mobile transport off.
func (s *Service) SetFCM(f *FCM) { s.fcm = f }

// Enabled reports whether VAPID keys are configured.
func (s *Service) Enabled() bool { return s.publicKey != "" && s.privateKey != "" }

// MobileEnabled reports whether pushes to the packaged app can be delivered.
func (s *Service) MobileEnabled() bool { return s.fcm != nil }

// PublicKey returns the VAPID public key for client subscription.
func (s *Service) PublicKey() string { return s.publicKey }

func (s *Service) Subscribe(ctx context.Context, userID uuid.UUID, sub Subscription) error {
	return s.repo.Upsert(ctx, s.pool, userID, sub)
}

func (s *Service) Unsubscribe(ctx context.Context, endpoint string) error {
	return s.repo.Delete(ctx, s.pool, endpoint)
}

// RegisterDevice stores the FCM registration token the mobile app reports.
func (s *Service) RegisterDevice(ctx context.Context, userID uuid.UUID, d Device) error {
	return s.repo.UpsertDevice(ctx, s.pool, userID, d)
}

// UnregisterDevice forgets a device, e.g. on sign-out.
func (s *Service) UnregisterDevice(ctx context.Context, token string) error {
	return s.repo.DeleteDevice(ctx, s.pool, token)
}

// payload is the JSON the service worker's push handler expects.
type payload struct {
	Title string `json:"title"`
	Body  string `json:"body"`
	URL   string `json:"url"`
	Tag   string `json:"tag,omitempty"`
}

// Notify pushes a notification to every device of a user: subscribed browsers
// and installed mobile apps. It runs its work synchronously; callers typically
// invoke it in a goroutine. Dead endpoints and tokens are pruned.
func (s *Service) Notify(ctx context.Context, userID uuid.UUID, title, body, url string) {
	s.notifyBrowsers(ctx, userID, title, body, url)
	s.notifyDevices(ctx, userID, title, body, url)
}

// notifyDevices delivers to the packaged mobile apps through Firebase.
func (s *Service) notifyDevices(ctx context.Context, userID uuid.UUID, title, body, url string) {
	if s.fcm == nil {
		return
	}
	devices, err := s.repo.ListDevicesForUser(ctx, s.pool, userID)
	if err != nil {
		s.log.Warn("push device list failed", "error", err)
		return
	}
	for _, d := range devices {
		switch err := s.fcm.Send(ctx, d.Token, title, body, url); {
		case err == nil:
		case errors.Is(err, ErrDeviceUnregistered):
			// The app is gone from that phone; stop paying for the round trip.
			if delErr := s.repo.DeleteDevice(ctx, s.pool, d.Token); delErr != nil {
				s.log.Warn("push device prune failed", "error", delErr)
			}
		default:
			s.log.Warn("push device send failed", "error", err)
		}
	}
}

func (s *Service) notifyBrowsers(ctx context.Context, userID uuid.UUID, title, body, url string) {
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
