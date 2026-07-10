// Package config loads and validates process configuration from
// environment variables. There are no default secrets: JWT keys and the
// database password must be supplied explicitly.
package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	Env      string
	LogLevel string

	HTTPPort int

	Postgres PostgresConfig
	Redis    RedisConfig

	JWTAccessSecret  string
	JWTRefreshSecret string
	JWTAccessTTL     time.Duration
	JWTRefreshTTL    time.Duration

	InvitationTokenTTL time.Duration

	// IPHashSalt salts IP digests stored in sessions and audit logs.
	IPHashSalt string

	// WSAllowedOrigin restricts WebSocket handshake origins in production.
	// Empty (development) allows any origin.
	WSAllowedOrigin string

	// VAPID keys enable Web Push. When the public/private pair is empty, push
	// is disabled. Subject is a mailto: or https: contact URL.
	VAPIDPublicKey  string
	VAPIDPrivateKey string
	VAPIDSubject    string

	// ICE configures WebRTC connectivity for 1:1 audio calls (STUN/TURN).
	ICE ICEConfig

	// Upload bounds attachment uploads (stage A: chunked uploads with
	// clearance-differentiated limits — no hardcoded 10 MiB).
	Upload UploadConfig

	// BootstrapCEOUsername/Password create the very first Level-1 account
	// when the users table is empty. Optional after first launch.
	BootstrapCEOUsername string
	BootstrapCEOPassword string

	// DatabaseURL / RedisURL, when set, override the individual connection
	// fields — used by managed-platform providers (Render, Neon, Upstash)
	// that inject a single connection string.
	DatabaseURL string
	RedisURL    string

	// WebDir, when set, makes the backend also serve the built SPA from
	// that directory (single-service same-origin deployment).
	WebDir string

	// RunMigrations forces schema migrations to run on boot. Defaults to
	// true outside production; a managed single-service deploy sets it true.
	RunMigrations bool
}

// PostgresDSN returns the effective database connection string.
func (c *Config) PostgresDSN() string {
	if c.DatabaseURL != "" {
		return c.DatabaseURL
	}
	return c.Postgres.DSN()
}

// ICEConfig holds the WebRTC ICE servers offered to clients via
// GET /calls/ice-config. STUNURLs are plain STUN endpoints; TURNURLs are relay
// endpoints whose short-lived credentials the backend derives from TURNSecret
// (coturn "static-auth-secret" / TURN REST API). An empty TURNSecret disables
// TURN and serves STUN only. Secrets never reach the client — only ephemeral,
// time-limited credentials do.
type ICEConfig struct {
	STUNURLs   []string
	TURNURLs   []string
	TURNSecret string
	TURNRealm  string
	TURNTTL    time.Duration
}

// UploadConfig bounds attachment uploads. Limits differ by clearance:
// leadership (levels 1–3) gets MaxBytesLeadership, everyone else
// MaxBytesStaff. ChunkBytes is the server-advertised chunk size for the
// chunked upload flow; SessionTTL reaps abandoned upload sessions.
type UploadConfig struct {
	MaxBytesLeadership int64
	MaxBytesStaff      int64
	ChunkBytes         int
	SessionTTL         time.Duration
}

// LeadershipMaxLevel is the strongest clearance band for upload limits:
// levels 1..3 (CEO, deputies, department heads) get the larger allowance.
const LeadershipMaxLevel = 3

// MaxBytesFor returns the upload ceiling for a clearance level.
func (u UploadConfig) MaxBytesFor(roleLevel int) int64 {
	if roleLevel <= LeadershipMaxLevel {
		return u.MaxBytesLeadership
	}
	return u.MaxBytesStaff
}

type PostgresConfig struct {
	Host     string
	Port     int
	Database string
	User     string
	Password string
	SSLMode  string
}

func (p PostgresConfig) DSN() string {
	return fmt.Sprintf(
		"postgres://%s:%s@%s:%d/%s?sslmode=%s",
		p.User, p.Password, p.Host, p.Port, p.Database, p.SSLMode,
	)
}

type RedisConfig struct {
	Host     string
	Port     int
	Password string
	DB       int
}

func (r RedisConfig) Addr() string {
	return fmt.Sprintf("%s:%d", r.Host, r.Port)
}

// Load reads configuration from the environment and fails fast if a
// required value is missing or malformed, rather than starting the
// process in a partially-configured state.
func Load() (*Config, error) {
	cfg := &Config{
		Env:      getEnv("APP_ENV", "development"),
		LogLevel: getEnv("LOG_LEVEL", "info"),
	}

	var err error

	// Managed platforms (Render, etc.) inject the listen port as PORT.
	if cfg.HTTPPort, err = getEnvInt("PORT", 0); err != nil {
		return nil, err
	}
	if cfg.HTTPPort == 0 {
		if cfg.HTTPPort, err = getEnvInt("BACKEND_HTTP_PORT", 8080); err != nil {
			return nil, err
		}
	}

	cfg.DatabaseURL = os.Getenv("DATABASE_URL")
	cfg.RedisURL = os.Getenv("REDIS_URL")

	pgPort, err := getEnvInt("POSTGRES_PORT", 5432)
	if err != nil {
		return nil, err
	}
	cfg.Postgres = PostgresConfig{
		Host:     getEnv("POSTGRES_HOST", "localhost"),
		Port:     pgPort,
		Database: getEnv("POSTGRES_DB", "kisy"),
		User:     getEnv("POSTGRES_USER", "kisy"),
		Password: os.Getenv("POSTGRES_PASSWORD"),
		SSLMode:  getEnv("POSTGRES_SSLMODE", "disable"),
	}
	// A discrete password is only required when no full DATABASE_URL is set.
	if cfg.DatabaseURL == "" && cfg.Postgres.Password == "" {
		return nil, fmt.Errorf("config: set DATABASE_URL or POSTGRES_PASSWORD")
	}

	redisPort, err := getEnvInt("REDIS_PORT", 6379)
	if err != nil {
		return nil, err
	}
	redisDB, err := getEnvInt("REDIS_DB", 0)
	if err != nil {
		return nil, err
	}
	cfg.Redis = RedisConfig{
		Host:     getEnv("REDIS_HOST", "localhost"),
		Port:     redisPort,
		Password: os.Getenv("REDIS_PASSWORD"),
		DB:       redisDB,
	}

	if cfg.JWTAccessSecret, err = requireEnvMinLen("JWT_ACCESS_SECRET", 32); err != nil {
		return nil, err
	}
	if cfg.JWTRefreshSecret, err = requireEnvMinLen("JWT_REFRESH_SECRET", 32); err != nil {
		return nil, err
	}
	if cfg.JWTAccessTTL, err = getEnvDuration("JWT_ACCESS_TTL", 15*time.Minute); err != nil {
		return nil, err
	}
	if cfg.JWTRefreshTTL, err = getEnvDuration("JWT_REFRESH_TTL", 720*time.Hour); err != nil {
		return nil, err
	}
	if cfg.InvitationTokenTTL, err = getEnvDuration("INVITATION_TOKEN_TTL", 120*time.Second); err != nil {
		return nil, err
	}

	// Falling back to the refresh secret keeps IP digests stable across
	// restarts without demanding one more mandatory variable.
	cfg.IPHashSalt = getEnv("IP_HASH_SALT", cfg.JWTRefreshSecret)

	cfg.BootstrapCEOUsername = os.Getenv("BOOTSTRAP_CEO_USERNAME")
	cfg.BootstrapCEOPassword = os.Getenv("BOOTSTRAP_CEO_PASSWORD")

	cfg.WSAllowedOrigin = os.Getenv("WS_ALLOWED_ORIGIN")
	cfg.VAPIDPublicKey = os.Getenv("VAPID_PUBLIC_KEY")
	cfg.VAPIDPrivateKey = os.Getenv("VAPID_PRIVATE_KEY")
	cfg.VAPIDSubject = getEnv("VAPID_SUBJECT", "mailto:admin@kisy.local")

	// ICE / WebRTC (audio calls). All optional: with no TURN configured the
	// client still gets STUN and works on non-symmetric NATs.
	cfg.ICE.STUNURLs = getEnvList("STUN_URLS")
	if len(cfg.ICE.STUNURLs) == 0 {
		// A public STUN default so NAT discovery works out of the box (e.g. on
		// Render, where no coturn is deployed). TURN still needs explicit
		// configuration for symmetric-NAT relaying.
		cfg.ICE.STUNURLs = []string{"stun:stun.l.google.com:19302"}
	}
	cfg.ICE.TURNURLs = getEnvList("TURN_URLS")
	cfg.ICE.TURNSecret = os.Getenv("TURN_SECRET")
	cfg.ICE.TURNRealm = getEnv("TURN_REALM", "kisy")
	if cfg.ICE.TURNTTL, err = getEnvDuration("TURN_TTL", 12*time.Hour); err != nil {
		return nil, err
	}

	// Attachment upload limits (stage A). Values are megabytes in env for
	// operator ergonomics; bytes internally.
	leadershipMB, err := getEnvInt("UPLOAD_MAX_MB_LEADERSHIP", 200)
	if err != nil {
		return nil, err
	}
	staffMB, err := getEnvInt("UPLOAD_MAX_MB_STAFF", 50)
	if err != nil {
		return nil, err
	}
	chunkKB, err := getEnvInt("UPLOAD_CHUNK_KB", 1024)
	if err != nil {
		return nil, err
	}
	cfg.Upload = UploadConfig{
		MaxBytesLeadership: int64(leadershipMB) << 20,
		MaxBytesStaff:      int64(staffMB) << 20,
		ChunkBytes:         chunkKB << 10,
	}
	if cfg.Upload.SessionTTL, err = getEnvDuration("UPLOAD_SESSION_TTL", 24*time.Hour); err != nil {
		return nil, err
	}

	cfg.WebDir = os.Getenv("WEB_DIR")

	// Migrations run automatically outside production; RUN_MIGRATIONS
	// overrides (managed single-service deploys set it true).
	cfg.RunMigrations = cfg.Env != "production"
	if v := os.Getenv("RUN_MIGRATIONS"); v != "" {
		cfg.RunMigrations = v == "true" || v == "1"
	}

	return cfg, nil
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// getEnvList parses a comma-separated environment variable into a trimmed,
// non-empty slice. An unset or empty variable yields nil.
func getEnvList(key string) []string {
	v := os.Getenv(key)
	if v == "" {
		return nil
	}
	var out []string
	for _, part := range strings.Split(v, ",") {
		if s := strings.TrimSpace(part); s != "" {
			out = append(out, s)
		}
	}
	return out
}

func requireEnv(key string) (string, error) {
	v := os.Getenv(key)
	if v == "" {
		return "", fmt.Errorf("config: required environment variable %s is not set", key)
	}
	return v, nil
}

func requireEnvMinLen(key string, minLen int) (string, error) {
	v, err := requireEnv(key)
	if err != nil {
		return "", err
	}
	if len(v) < minLen {
		return "", fmt.Errorf("config: %s must be at least %d characters", key, minLen)
	}
	return v, nil
}

func getEnvInt(key string, fallback int) (int, error) {
	v := os.Getenv(key)
	if v == "" {
		return fallback, nil
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return 0, fmt.Errorf("config: %s must be an integer: %w", key, err)
	}
	return n, nil
}

func getEnvDuration(key string, fallback time.Duration) (time.Duration, error) {
	v := os.Getenv(key)
	if v == "" {
		return fallback, nil
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		return 0, fmt.Errorf("config: %s must be a valid duration: %w", key, err)
	}
	return d, nil
}
