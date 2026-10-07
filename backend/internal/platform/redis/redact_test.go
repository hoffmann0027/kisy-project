package redis

import (
	"context"
	"strings"
	"testing"

	"kisy-backend/internal/config"
)

// Audit A-39: go-redis quotes a REDIS_URL it cannot parse, password included.
func TestAMalformedURLDoesNotCarryThePassword(t *testing.T) {
	const password = "SuperSecretPW1"
	_, err := NewClient(context.Background(), config.RedisConfig{}, "rediss://default:"+password+"@host:notaport")
	if err == nil || strings.Contains(err.Error(), password) {
		t.Fatalf("redis: %v", err)
	}
}
