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
	"time"

	webpush "github.com/SherClockHolmes/webpush-go"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"kisy-backend/internal/i18n"
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
	// DeleteForUser removes a subscription only if it belongs to userID.
	DeleteForUser(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID, endpoint string) error
	ListForUser(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID) ([]Subscription, error)

	UpsertDevice(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID, d Device) error
	DeleteDevice(ctx context.Context, pool *pgxpool.Pool, token string) error
	// DeleteDeviceForUser removes a device token only if it belongs to userID.
	DeleteDeviceForUser(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID, token string) error
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

func (r *PostgresRepository) DeleteForUser(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID, endpoint string) error {
	if _, err := pool.Exec(ctx, `DELETE FROM push_subscriptions WHERE endpoint = $1 AND user_id = $2`, endpoint, userID); err != nil {
		return fmt.Errorf("push: delete own: %w", err)
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
//
// That is deliberate, and it is the one half of audit A-20 kept as it was: the
// token is the device's own secret, so presenting it is presenting the device,
// and notifications belong to whoever holds the phone now. Refusing the rebind
// would leave the previous account's notifications on a phone handed to
// someone else. Deleting, on the other hand, has no such case — see
// DeleteDeviceForUser.
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

func (r *PostgresRepository) DeleteDeviceForUser(ctx context.Context, pool *pgxpool.Pool, userID uuid.UUID, token string) error {
	if _, err := pool.Exec(ctx, `DELETE FROM device_tokens WHERE token = $1 AND user_id = $2`, token, userID); err != nil {
		return fmt.Errorf("push: delete own device: %w", err)
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

	// web is the client browser pushes go through (endpoint.go).
	web *http.Client

	// localeOf finds the language a push is written in: the recipient's.
	localeOf func(context.Context, uuid.UUID) i18n.Lang
}

// SetLocaleOf wires how a recipient's language is found. Unset, every push is
// in i18n.Default.
func (s *Service) SetLocaleOf(f func(context.Context, uuid.UUID) i18n.Lang) { s.localeOf = f }

func NewService(pool *pgxpool.Pool, repo Repository, log *slog.Logger, publicKey, privateKey, subject string) *Service {
	return &Service{
		pool: pool, repo: repo, log: log, publicKey: publicKey, privateKey: privateKey, subject: subject,
		web: newSendClient(),
	}
}

// notifyTimeout bounds one whole fan-out — every browser and every device of
// one person. Callers start Notify in a goroutine on context.Background() so
// the push outlives the request that caused it; this is what keeps that
// goroutine from living forever when a push service hangs.
const notifyTimeout = 30 * time.Second

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
	if err := ValidateEndpoint(sub.Endpoint); err != nil {
		return err
	}
	return s.repo.Upsert(ctx, s.pool, userID, sub)
}

// Unsubscribe removes the caller's own subscription. It used to delete any
// row with that endpoint, so whoever knew someone's endpoint could switch off
// their notifications (audit A-20).
func (s *Service) Unsubscribe(ctx context.Context, userID uuid.UUID, endpoint string) error {
	return s.repo.DeleteForUser(ctx, s.pool, userID, endpoint)
}

// RegisterDevice stores the FCM registration token the mobile app reports.
func (s *Service) RegisterDevice(ctx context.Context, userID uuid.UUID, d Device) error {
	return s.repo.UpsertDevice(ctx, s.pool, userID, d)
}

// UnregisterDevice forgets a device, e.g. on sign-out.
// UnregisterDevice removes the caller's own device token (audit A-20, as
// Unsubscribe). Tokens a push service reports as dead are still pruned
// whoever they belong to — that is the service's word, not a client's.
func (s *Service) UnregisterDevice(ctx context.Context, userID uuid.UUID, token string) error {
	return s.repo.DeleteDeviceForUser(ctx, s.pool, userID, token)
}

// payload is the JSON the service worker's push handler expects.
type payload struct {
	Title string `json:"title"`
	Body  string `json:"body"`
	URL   string `json:"url"`
	Tag   string `json:"tag,omitempty"`
}

// DefaultTag is the tag most pushes share: each new one replaces the last
// instead of piling up.
const DefaultTag = "kisy"

// RetractType is the data message that tells the app to take a shown push
// down. The Android side (CallPushService.java) matches on it.
const RetractType = "notification_retract"

// RetractTTL is how long FCM keeps trying to deliver a retraction: as long as
// it keeps the push it takes back (the four-week default), so a phone that
// was off the whole time gets both, not just the push.
const RetractTTL = 28 * 24 * time.Hour

// Notify pushes a notification to every device of a user: subscribed browsers
// and installed mobile apps, worded in that user's language. It runs its work
// synchronously; callers typically invoke it in a goroutine. Dead endpoints
// and tokens are pruned.
func (s *Service) Notify(ctx context.Context, userID uuid.UUID, title, body i18n.Msg, url string) {
	s.NotifyTagged(ctx, userID, DefaultTag, title, body, url)
}

// NotifyTagged is Notify under its own tag, for a push that may have to be
// taken back with Retract — and that a later chat message must not replace.
func (s *Service) NotifyTagged(ctx context.Context, userID uuid.UUID, tag string, title, body i18n.Msg, url string) {
	ctx, cancel := context.WithTimeout(ctx, notifyTimeout)
	defer cancel()
	lang := i18n.Default
	if s.localeOf != nil {
		lang = s.localeOf(ctx, userID)
	}
	t, b := title.In(lang), body.In(lang)
	s.notifyBrowsers(ctx, userID, tag, t, b, url)
	s.notifyDevices(ctx, userID, tag, t, b, url)
}

// Retract takes down a push shown under tag on the user's phones. A push
// drawn by Android stays in the shade until something removes it — deleting
// the notification it announced does not — so the app is sent a data message
// that it answers by cancelling the tag.
//
// Browsers are not sent anything: a web push that shows nothing is answered
// by the browser with a generic notice of its own. The open app closes the
// browser's copy instead (frontend shared/lib/shownNotifications.ts).
func (s *Service) Retract(ctx context.Context, userID uuid.UUID, tag string) {
	ctx, cancel := context.WithTimeout(ctx, notifyTimeout)
	defer cancel()
	s.SendData(ctx, userID, map[string]string{"type": RetractType, "tag": tag}, RetractTTL)
}

// CallInviteTTL bounds how long FCM keeps trying to deliver a ring. Past it
// the caller has long given up, and a phone that rings for a call nobody is
// making is worse than a missed one. It also sets the floor for how long the
// callee's side may take before the call is written off as missed.
const CallInviteTTL = 45 * time.Second

// HasDevices reports whether the user has any registered mobile device.
// Used to decide whether an offline callee is merely asleep or genuinely
// unreachable, without spending a message to find out.
func (s *Service) HasDevices(ctx context.Context, userID uuid.UUID) bool {
	if s.fcm == nil {
		return false
	}
	devices, err := s.repo.ListDevicesForUser(ctx, s.pool, userID)
	if err != nil {
		s.log.Warn("push device list failed", "error", err)
		return false
	}
	return len(devices) > 0
}

// SendData delivers a data-only message to every device of one user.
//
// Data-only on purpose: a notification payload is drawn by Android itself and
// the app is never started, so a swiped-away app could not ring. Here the app
// is woken and decides what to show. Returns whether at least one device
// accepted it — the caller uses that to tell "nobody has this app installed"
// from "the phone is merely asleep".
func (s *Service) SendData(ctx context.Context, userID uuid.UUID, data map[string]string, ttl time.Duration) bool {
	if s.fcm == nil {
		return false
	}
	devices, err := s.repo.ListDevicesForUser(ctx, s.pool, userID)
	if err != nil {
		s.log.Warn("push device list failed", "error", err)
		return false
	}
	delivered := false
	for _, d := range devices {
		switch err := s.fcm.SendData(ctx, d.Token, data, ttl); {
		case err == nil:
			delivered = true
		case errors.Is(err, ErrDeviceUnregistered):
			if delErr := s.repo.DeleteDevice(ctx, s.pool, d.Token); delErr != nil {
				s.log.Warn("push device prune failed", "error", delErr)
			}
		default:
			s.log.Warn("push data send failed", "error", err)
		}
	}
	return delivered
}

// notifyDevices delivers to the packaged mobile apps through Firebase.
func (s *Service) notifyDevices(ctx context.Context, userID uuid.UUID, tag, title, body, url string) {
	if s.fcm == nil {
		return
	}
	devices, err := s.repo.ListDevicesForUser(ctx, s.pool, userID)
	if err != nil {
		s.log.Warn("push device list failed", "error", err)
		return
	}
	for _, d := range devices {
		switch err := s.fcm.SendTagged(ctx, d.Token, tag, title, body, url); {
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

func (s *Service) notifyBrowsers(ctx context.Context, userID uuid.UUID, tag, title, body, url string) {
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
	data, _ := json.Marshal(payload{Title: title, Body: body, URL: url, Tag: tag})

	for _, sub := range subs {
		// Rows stored before endpoints were checked may point anywhere: they
		// are dropped rather than called (audit A-17).
		if ValidateEndpoint(sub.Endpoint) != nil {
			_ = s.repo.Delete(ctx, s.pool, sub.Endpoint)
			continue
		}
		resp, err := webpush.SendNotificationWithContext(ctx, data, &webpush.Subscription{
			Endpoint: sub.Endpoint,
			Keys:     webpush.Keys{P256dh: sub.P256dh, Auth: sub.Auth},
		}, &webpush.Options{
			Subscriber:      s.subject,
			VAPIDPublicKey:  s.publicKey,
			VAPIDPrivateKey: s.privateKey,
			TTL:             86400,
			HTTPClient:      s.web,
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
