package config

import (
	"strings"
	"testing"
)

// TestEveryBurnedSecretIsCompared pins half of audit D-08: the prod guard that
// refuses known-compromised secrets is what .gitleaksignore relies on, yet
// nothing tested it. Deleting either the digest or the variable from the
// comparison would have gone through CI unnoticed — and the ignored finding
// would then point at a check that no longer exists.
func TestEveryBurnedSecretIsCompared(t *testing.T) {
	if len(knownCompromisedDigests) == 0 {
		t.Fatal("the burned-secret table is empty: the prod guard .gitleaksignore relies on is gone")
	}

	cfg := &Config{}
	compared := cfg.checkedSecrets()

	for digest, origin := range knownCompromisedDigests {
		if len(digest) != 64 {
			t.Errorf("%s: digest is not a SHA-256 hex string: %q", origin, digest)
		}
		// "JWT_ACCESS_SECRET (July 2026 audit bundle)" → "JWT_ACCESS_SECRET".
		name := origin
		if i := strings.IndexByte(origin, ' '); i > 0 {
			name = origin[:i]
		}
		if _, ok := compared[name]; !ok {
			t.Errorf("%s is listed as burned but %s is never compared against the table", origin, name)
		}
	}
}

// TestCompromisedValueRefusedWithSurroundingWhitespace pins the other half:
// the comparison used the raw value, so the same burned key pasted into a
// dashboard with a trailing newline sailed straight past it.
func TestCompromisedValueRefusedWithSurroundingWhitespace(t *testing.T) {
	const burned = "burned-secret-value-for-the-test"
	digest := sha256Hex(burned)
	knownCompromisedDigests[digest] = "JWT_ACCESS_SECRET (test fixture)"
	t.Cleanup(func() { delete(knownCompromisedDigests, digest) })

	for _, value := range []string{burned, burned + "\n", "  " + burned + " \t"} {
		setBaseEnv(t)
		t.Setenv("APP_ENV", "production")
		t.Setenv("POSTGRES_SSLMODE", "require")
		t.Setenv("JWT_ACCESS_SECRET", value)

		_, err := Load()
		if err == nil {
			t.Fatalf("production started with a burned secret (value %q)", value)
		}
		if !strings.Contains(err.Error(), "known-compromised") {
			t.Fatalf("value %q: want a known-compromised refusal, got %v", value, err)
		}
	}
}
