// Package redis wires the shared Redis client used for caching, presence
// and session-adjacent state.
package redis

import (
	"context"
	"fmt"

	"github.com/redis/go-redis/v9"

	"kisy-backend/internal/config"
)

// NewClient creates a client and verifies connectivity with a ping. When
// url is non-empty (e.g. a managed Redis/Key-Value connection string,
// possibly rediss:// for TLS) it takes precedence over the discrete fields.
func NewClient(ctx context.Context, cfg config.RedisConfig, url string) (*redis.Client, error) {
	var opts *redis.Options
	if url != "" {
		parsed, err := redis.ParseURL(url)
		if err != nil {
			return nil, fmt.Errorf("redis: parse REDIS_URL: %w", err)
		}
		opts = parsed
	} else {
		opts = &redis.Options{Addr: cfg.Addr(), Password: cfg.Password, DB: cfg.DB}
	}

	client := redis.NewClient(opts)
	if err := client.Ping(ctx).Err(); err != nil {
		_ = client.Close()
		return nil, fmt.Errorf("redis: ping: %w", err)
	}
	return client, nil
}
