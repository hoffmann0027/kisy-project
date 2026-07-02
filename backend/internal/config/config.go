// Package config loads and validates process configuration from
// environment variables. There are no default secrets: JWT keys and the
// database password must be supplied explicitly.
package config

import (
	"fmt"
	"os"
	"strconv"
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

	if cfg.HTTPPort, err = getEnvInt("BACKEND_HTTP_PORT", 8080); err != nil {
		return nil, err
	}

	pgPort, err := getEnvInt("POSTGRES_PORT", 5432)
	if err != nil {
		return nil, err
	}
	cfg.Postgres = PostgresConfig{
		Host:     getEnv("POSTGRES_HOST", "localhost"),
		Port:     pgPort,
		Database: getEnv("POSTGRES_DB", "kisy"),
		User:     getEnv("POSTGRES_USER", "kisy"),
		SSLMode:  getEnv("POSTGRES_SSLMODE", "disable"),
	}
	if cfg.Postgres.Password, err = requireEnv("POSTGRES_PASSWORD"); err != nil {
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

	return cfg, nil
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
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
