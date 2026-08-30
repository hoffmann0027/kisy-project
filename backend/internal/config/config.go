// Package config loads and validates process configuration from
// environment variables. There are no default secrets: JWT keys and the
// database password must be supplied explicitly.
package config

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"net/netip"
	"net/url"
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
	DBPool   DBPoolConfig
	Redis    RedisConfig

	JWTAccessSecret  string
	JWTRefreshSecret string
	JWTAccessTTL     time.Duration
	JWTRefreshTTL    time.Duration

	InvitationTokenTTL time.Duration

	// IPHashSalt salts IP digests stored in sessions and audit logs.
	IPHashSalt string

	// NativeAppOrigins are the WebView origins of the packaged mobile apps
	// (Capacitor serves the bundle from https://localhost on Android and
	// capacitor://localhost on iOS). They are allowed through the CSRF and
	// WebSocket origin checks; those clients use Bearer tokens, not cookies.
	// Set NATIVE_APP_ORIGINS="" to refuse them entirely.
	NativeAppOrigins []string

	// WSAllowedOrigin restricts WebSocket handshake origins in production.
	// Empty (development) allows any origin.
	WSAllowedOrigin string

	// TrustedProxyCIDRs lists the networks our own reverse proxies live in.
	// Only when a hop's address falls inside one of them is its
	// X-Forwarded-For entry believed; empty (the default) means no header is
	// trusted at all and the TCP peer address is the client. Getting this
	// wrong in the permissive direction lets a caller forge its own IP and
	// escape every per-IP rate limit, so it is opt-in and explicit.
	TrustedProxyCIDRs []string

	// VAPID keys enable Web Push. When the public/private pair is empty, push
	// is disabled. Subject is a mailto: or https: contact URL.
	VAPIDPublicKey  string
	VAPIDPrivateKey string
	VAPIDSubject    string

	// FCMServiceAccount is a Firebase service-account key file (the raw JSON,
	// or the same JSON base64-encoded for environments that dislike newlines).
	// Empty disables push to the packaged mobile app.
	FCMServiceAccount []byte

	// ICE configures WebRTC connectivity for 1:1 audio calls (STUN/TURN).
	ICE ICEConfig

	// Blob selects where uploaded file bytes live (O5). Empty = Postgres.
	Blob BlobConfig

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

// BlobConfig points at the S3-compatible object store that holds uploaded file
// bytes (attachments, avatars). Leave Endpoint/Bucket empty to keep storing
// bytes in Postgres — reads always fall back to the database, so switching it
// on is safe before any migration has run.
type BlobConfig struct {
	Endpoint  string
	Bucket    string
	AccessKey string
	SecretKey string
	Region    string
	UseSSL    bool
}

// Enabled reports whether uploads should go to object storage.
func (b BlobConfig) Enabled() bool { return b.Endpoint != "" && b.Bucket != "" }

type PostgresConfig struct {
	Host     string
	Port     int
	Database string
	User     string
	Password string
	SSLMode  string
}

// DBPoolConfig tunes the pgx connection pool. Defaults are sized for a single
// small instance against a managed Postgres: MaxConns must leave headroom under
// the server's max_connections once every backend replica is counted (see O6),
// and MaxConnIdleTime keeps idle connections from pinning a serverless
// compute (Neon) awake — or from being severed server-side without our notice.
type DBPoolConfig struct {
	MaxConns          int32
	MinConns          int32
	MaxConnLifetime   time.Duration
	MaxConnIdleTime   time.Duration
	HealthCheckPeriod time.Duration
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

	if cfg.DBPool, err = loadDBPool(); err != nil {
		return nil, err
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
	cfg.NativeAppOrigins = splitList(getEnvDefault("NATIVE_APP_ORIGINS", "https://localhost,capacitor://localhost"))

	// Validated here rather than at first use: the router would otherwise
	// panic mid-startup on a typo, and a silently dropped prefix would mean
	// silently trusting nobody (or, worse, the wrong hop).
	cfg.TrustedProxyCIDRs = getEnvList("TRUSTED_PROXY_CIDRS")
	for _, c := range cfg.TrustedProxyCIDRs {
		if _, err := netip.ParsePrefix(c); err != nil {
			return nil, fmt.Errorf("config: TRUSTED_PROXY_CIDRS entry %q is not a CIDR prefix (e.g. 172.16.0.0/12)", c)
		}
	}
	cfg.VAPIDPublicKey = os.Getenv("VAPID_PUBLIC_KEY")
	cfg.VAPIDPrivateKey = os.Getenv("VAPID_PRIVATE_KEY")
	cfg.VAPIDSubject = getEnv("VAPID_SUBJECT", "mailto:admin@kisy.local")
	cfg.FCMServiceAccount = decodeServiceAccount(os.Getenv("FCM_SERVICE_ACCOUNT"))

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

	// Object storage for uploaded bytes (O5). Unset → bytes stay in Postgres.
	cfg.Blob = BlobConfig{
		Endpoint:  os.Getenv("BLOB_S3_ENDPOINT"),
		Bucket:    os.Getenv("BLOB_S3_BUCKET"),
		AccessKey: os.Getenv("BLOB_S3_ACCESS_KEY"),
		SecretKey: os.Getenv("BLOB_S3_SECRET_KEY"),
		Region:    os.Getenv("BLOB_S3_REGION"),
		UseSSL:    os.Getenv("BLOB_S3_USE_SSL") == "true",
	}

	cfg.WebDir = os.Getenv("WEB_DIR")

	// Migrations run automatically outside production; RUN_MIGRATIONS
	// overrides (managed single-service deploys set it true).
	cfg.RunMigrations = cfg.Env != "production"
	if v := os.Getenv("RUN_MIGRATIONS"); v != "" {
		cfg.RunMigrations = v == "true" || v == "1"
	}

	if err := cfg.validateProduction(); err != nil {
		return nil, err
	}

	return cfg, nil
}

// knownCompromisedDigests maps SHA-256 digests of secret values that have
// left the trusted environment (they were bundled into an archive during the
// July 2026 security audit and must be treated as burned) to the variable
// they were used for. Production refuses to boot with any of them. Storing
// digests of dead 256-bit-entropy values reveals nothing useful.
var knownCompromisedDigests = map[string]string{
	"9fd557efaff4f461a536b055f59aa9eee380e569856f4d0ead606876866c3968": "POSTGRES_PASSWORD (July 2026 audit bundle)",
	"741ef7367130c8263f39d8e7d72ff22ae9a4d86bd98fae9b2024e0adc5e82c2a": "REDIS_PASSWORD (July 2026 audit bundle)",
	"bd7fd072deef8ccb0f6bdc680aeb9e5b11dc9d3f2ee7c973b7054df9bb36f0c6": "JWT_ACCESS_SECRET (July 2026 audit bundle)",
	"9eedc2dc2772d95e0c9903d8a7bc2a8e41ebe479236a377d6e166ca7ff6eda49": "JWT_REFRESH_SECRET (July 2026 audit bundle)",
	"eb67cc63803f41dbcfd4ff780ebcf44e984e8c5654ef2a18ceebcf7b9729eb9f": "IP_HASH_SALT (July 2026 audit bundle)",
	"332ebc8c4d084d42fe6405fdc6a4bd323073d254253ced28c9917dcd0b08c9e8": "BOOTSTRAP_CEO_PASSWORD (July 2026 audit bundle)",
	"6c31a12221c3e378d89d4824a5fc7d87f11bebe6486b21ad0788f5336a0f12a9": "TURN_SECRET (July 2026 audit bundle)",
	"c56b5c4f856a1beb62b0c658b5f34f76d0590e2e183fba32b9761e8d90c6f1af": "VAPID_PRIVATE_KEY (dev pair committed in .env.example history)",
}

// loadDBPool reads the pgx pool knobs from the environment.
//
// Defaults: 10 connections is comfortably under a managed Postgres' limit
// (Neon free allows ~100) while leaving room for a second replica plus admin
// sessions; 2 idle connections keep the common path warm without holding a
// serverless compute open; a 30-minute lifetime recycles connections so a
// rotated credential or a moved endpoint cannot pin a stale one forever.
func loadDBPool() (DBPoolConfig, error) {
	maxConns, err := getEnvInt("DB_POOL_MAX_CONNS", 10)
	if err != nil {
		return DBPoolConfig{}, err
	}
	minConns, err := getEnvInt("DB_POOL_MIN_CONNS", 2)
	if err != nil {
		return DBPoolConfig{}, err
	}
	lifetime, err := getEnvDuration("DB_POOL_MAX_CONN_LIFETIME", 30*time.Minute)
	if err != nil {
		return DBPoolConfig{}, err
	}
	idle, err := getEnvDuration("DB_POOL_MAX_CONN_IDLE_TIME", 5*time.Minute)
	if err != nil {
		return DBPoolConfig{}, err
	}
	health, err := getEnvDuration("DB_POOL_HEALTHCHECK_PERIOD", time.Minute)
	if err != nil {
		return DBPoolConfig{}, err
	}

	if maxConns < 1 {
		return DBPoolConfig{}, fmt.Errorf("config: DB_POOL_MAX_CONNS must be >= 1")
	}
	if minConns < 0 || minConns > maxConns {
		return DBPoolConfig{}, fmt.Errorf("config: DB_POOL_MIN_CONNS must be between 0 and DB_POOL_MAX_CONNS")
	}

	return DBPoolConfig{
		MaxConns:          int32(maxConns),
		MinConns:          int32(minConns),
		MaxConnLifetime:   lifetime,
		MaxConnIdleTime:   idle,
		HealthCheckPeriod: health,
	}, nil
}

func sha256Hex(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])
}

// validateProduction is the fail-closed gate for APP_ENV=production: an
// insecure configuration must refuse to start rather than silently run. It
// collects every violation so the operator can fix them in one pass.
func (c *Config) validateProduction() error {
	if c.Env != "production" {
		return nil
	}

	var problems []string

	// The WebSocket handshake and CSRF middleware are cookie-authenticated;
	// without an explicit allowed origin only same-host requests pass, and an
	// unset value usually means the operator forgot to configure it.
	// A single-service deploy that also serves the SPA (WebDir set) is
	// same-origin, so the WS/CSRF same-host fallback is correct and secure —
	// an explicit WS_ALLOWED_ORIGIN is optional there. For a split deployment
	// (API on its own origin) an empty value usually signals a misconfiguration
	// and must be set.
	if c.WSAllowedOrigin == "" && c.WebDir == "" {
		problems = append(problems, "WS_ALLOWED_ORIGIN must be set (e.g. https://kisy.example) — WebSocket/CSRF origin allowlist")
	}

	// Plaintext to the database is only acceptable over loopback (same-host /
	// all-in-one deployments). A private single-host docker network may opt in
	// explicitly with ALLOW_PLAINTEXT_DB=true; anything else requires TLS
	// (sslmode=require, better verify-full).
	if c.insecureDBTransport() && os.Getenv("ALLOW_PLAINTEXT_DB") != "true" {
		problems = append(problems, "POSTGRES_SSLMODE=disable over a non-loopback connection — enable TLS to the database, or set ALLOW_PLAINTEXT_DB=true only for a single-host private docker network")
	}

	if c.RedisURL == "" && c.Redis.Password == "" {
		problems = append(problems, "REDIS_PASSWORD must be set (or provide REDIS_URL with credentials)")
	}

	// Secrets that ever left the trusted environment are burned: refuse them.
	for name, val := range map[string]string{
		"JWT_ACCESS_SECRET":      c.JWTAccessSecret,
		"JWT_REFRESH_SECRET":     c.JWTRefreshSecret,
		"POSTGRES_PASSWORD":      c.Postgres.Password,
		"REDIS_PASSWORD":         c.Redis.Password,
		"IP_HASH_SALT":           c.IPHashSalt,
		"TURN_SECRET":            c.ICE.TURNSecret,
		"BOOTSTRAP_CEO_PASSWORD": c.BootstrapCEOPassword,
		"VAPID_PRIVATE_KEY":      c.VAPIDPrivateKey,
	} {
		if val == "" {
			continue
		}
		if origin, burned := knownCompromisedDigests[sha256Hex(val)]; burned {
			problems = append(problems, fmt.Sprintf("%s is a known-compromised value (%s) — rotate it", name, origin))
		}
	}

	if len(problems) > 0 {
		return fmt.Errorf("config: refusing to start in production:\n  - %s", strings.Join(problems, "\n  - "))
	}
	return nil
}

// insecureDBTransport reports whether the effective database connection would
// travel in plaintext over a non-loopback transport.
func (c *Config) insecureDBTransport() bool {
	if c.DatabaseURL != "" {
		u, err := url.Parse(c.DatabaseURL)
		if err != nil {
			return false // the pool constructor will surface the real error
		}
		// Only an explicit sslmode=disable is insecure; drivers default to
		// negotiating TLS ("prefer") and managed platforms terminate TLS.
		if u.Query().Get("sslmode") != "disable" {
			return false
		}
		return !isLoopbackHost(u.Hostname())
	}
	if c.Postgres.SSLMode != "disable" {
		return false
	}
	return !isLoopbackHost(c.Postgres.Host)
}

func isLoopbackHost(host string) bool {
	switch strings.ToLower(host) {
	case "", "localhost", "127.0.0.1", "::1":
		// An empty host means a unix socket — no network transport at all.
		return true
	default:
		return false
	}
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

// getEnvDefault returns the environment value, or fallback when the variable
// is not set at all. An explicitly empty value is honoured (it means "none").
func getEnvDefault(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok {
		return v
	}
	return fallback
}

// splitList parses a comma-separated env value into trimmed, non-empty items.
// decodeServiceAccount accepts the Firebase key either as raw JSON or as
// base64. Hosting dashboards mangle multi-line values often enough that the
// encoded form is the practical one, but a pasted file should work too.
func decodeServiceAccount(v string) []byte {
	v = strings.TrimSpace(v)
	if v == "" {
		return nil
	}
	if strings.HasPrefix(v, "{") {
		return []byte(v)
	}
	decoded, err := base64.StdEncoding.DecodeString(v)
	if err != nil {
		// Hand the raw bytes on: push.NewFCM reports precisely what is wrong.
		return []byte(v)
	}
	return decoded
}

func splitList(v string) []string {
	var out []string
	for _, part := range strings.Split(v, ",") {
		if s := strings.TrimSpace(part); s != "" {
			out = append(out, s)
		}
	}
	return out
}
