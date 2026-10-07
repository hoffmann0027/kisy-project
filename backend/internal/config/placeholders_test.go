package config

import (
	"strings"
	"testing"
)

// Audit A-24: production started with values that are public. The
// .env.example placeholders live in the repository, and the JWT one is 37
// characters long, so it passed the 32-character check — a compose deploy
// that copied the template signed its tokens with a key anyone could read.
func TestProductionRefusesPublicOrDegenerateSecrets(t *testing.T) {
	cases := []struct {
		name string
		env  map[string]string
		want string
	}{
		{"jwt placeholder", map[string]string{"JWT_ACCESS_SECRET": "change_me_min_32_chars_______________"}, "JWT_ACCESS_SECRET is still the placeholder"},
		{"db placeholder", map[string]string{"POSTGRES_PASSWORD": "change_me"}, "POSTGRES_PASSWORD is still the placeholder"},
		{"ceo placeholder", map[string]string{"BOOTSTRAP_CEO_PASSWORD": "change_me_ceo_password_123"}, "BOOTSTRAP_CEO_PASSWORD is still the placeholder"},
		{"placeholder in capitals", map[string]string{"TURN_SECRET": "CHANGE_ME_turn", "TURN_URLS": "turn:t.example:3478"}, "TURN_SECRET is still the placeholder"},
		{"same secret for both tokens", map[string]string{
			"JWT_ACCESS_SECRET": strings.Repeat("x", 40), "JWT_REFRESH_SECRET": strings.Repeat("x", 40),
		}, "must differ"},
		{"redis url without password", map[string]string{"REDIS_URL": "redis://cache.internal:6379/0"}, "REDIS_URL has no password"},
		{"redis url with a user but no password", map[string]string{"REDIS_URL": "rediss://default@cache.example:6379"}, "REDIS_URL has no password"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			setBaseEnv(t)
			t.Setenv("APP_ENV", "production")
			t.Setenv("POSTGRES_SSLMODE", "require")
			for k, v := range tc.env {
				t.Setenv(k, v)
			}
			_, err := Load()
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("want a refusal containing %q, got %v", tc.want, err)
			}
		})
	}
}

func TestProductionAcceptsRealRedisURLs(t *testing.T) {
	for _, url := range []string{
		"rediss://default:s3cr3t@eu1-sharp-fox-12345.upstash.io:6379", // Upstash, as on Render
		"redis://localhost:6379/0",                                    // loopback needs no password
	} {
		setBaseEnv(t)
		t.Setenv("APP_ENV", "production")
		t.Setenv("POSTGRES_SSLMODE", "require")
		t.Setenv("REDIS_URL", url)
		if _, err := Load(); err != nil {
			t.Fatalf("REDIS_URL %q refused: %v", url, err)
		}
	}
}
