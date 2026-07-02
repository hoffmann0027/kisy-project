// Package redis wires the shared Redis client used for caching, presence
// and session-adjacent state.
package redis

import (
	"context"
	"fmt"

	"github.com/redis/go-redis/v9"

	"kisy-backend/internal/config"
)

// NewClient creates a client and verifies connectivity with a ping.
func NewClient(ctx context.Context, cfg config.RedisConfig) (*redis.Client, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     cfg.Addr(),
		Password: cfg.Password,
		DB:       cfg.DB,
	})

	if err := client.Ping(ctx).Err(); err != nil {
		_ = client.Close()
		return nil, fmt.Errorf("redis: ping: %w", err)
	}

	return client, nil
}
