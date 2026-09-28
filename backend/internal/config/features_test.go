package config

import (
	"strings"
	"testing"
)

// TestProductionRefusesHalfConfiguredFeatures pins audit D-03: a feature with
// half its variables filled in used to switch itself off in silence. Every
// case below produced a deploy that reported itself healthy while web push,
// relayed calls or object storage were dead.
func TestProductionRefusesHalfConfiguredFeatures(t *testing.T) {
	cases := []struct {
		name string
		env  map[string]string
		want string
	}{
		{
			name: "vapid public key without private",
			env:  map[string]string{"VAPID_PUBLIC_KEY": "public-half"},
			want: "VAPID_PUBLIC_KEY and VAPID_PRIVATE_KEY",
		},
		{
			name: "vapid private key without public",
			env:  map[string]string{"VAPID_PRIVATE_KEY": "private-half"},
			want: "VAPID_PUBLIC_KEY and VAPID_PRIVATE_KEY",
		},
		{
			name: "turn urls without secret",
			env:  map[string]string{"TURN_URLS": "turn:turn.kisy.example:3478"},
			want: "TURN_URLS is set without TURN_SECRET",
		},
		{
			name: "turn secret without urls",
			env:  map[string]string{"TURN_SECRET": "relay-secret-value"},
			want: "TURN_SECRET is set without TURN_URLS",
		},
		{
			name: "object storage without credentials",
			env: map[string]string{
				"BLOB_S3_ENDPOINT": "https://s3.example",
				"BLOB_S3_BUCKET":   "kisy-files",
			},
			want: "BLOB_S3_ACCESS_KEY, BLOB_S3_SECRET_KEY",
		},
		{
			name: "object storage endpoint alone",
			env:  map[string]string{"BLOB_S3_ENDPOINT": "https://s3.example"},
			want: "object storage is half configured",
		},
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
			if err == nil {
				t.Fatal("production started with a half-configured feature")
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("error does not name the problem: want %q in\n%v", tc.want, err)
			}
		})
	}
}

// TestCompleteFeatureConfigurationStarts guards the other direction: a
// complete pair, and no pair at all, must both boot. The check exists to
// catch omissions, not to make optional features mandatory.
func TestCompleteFeatureConfigurationStarts(t *testing.T) {
	setBaseEnv(t)
	t.Setenv("APP_ENV", "production")
	t.Setenv("POSTGRES_SSLMODE", "require")
	t.Setenv("VAPID_PUBLIC_KEY", "public-half")
	t.Setenv("VAPID_PRIVATE_KEY", "private-half")
	t.Setenv("TURN_URLS", "turn:turn.kisy.example:3478")
	t.Setenv("TURN_SECRET", "relay-secret-value")
	t.Setenv("BLOB_S3_ENDPOINT", "https://s3.example")
	t.Setenv("BLOB_S3_BUCKET", "kisy-files")
	t.Setenv("BLOB_S3_ACCESS_KEY", "access")
	t.Setenv("BLOB_S3_SECRET_KEY", "secret")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("fully configured production load: %v", err)
	}

	f := cfg.Features()
	if !f.WebPush || !f.TURN || !f.BlobS3 {
		t.Fatalf("features misreported: %s", f)
	}
	if got := f.String(); !strings.Contains(got, "webpush=on") || !strings.Contains(got, "blob_s3=on") {
		t.Fatalf("summary = %q", got)
	}
	if f.Map()["fcm"] {
		t.Fatal("fcm reported on without a service account")
	}
}
