// Package password implements Argon2id password hashing using the PHC
// string format, per docs/spec/06-security.md ("Argon2id with strong
// parameters").
package password

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"

	"golang.org/x/crypto/argon2"
)

// OWASP-recommended Argon2id parameters (64 MiB, 3 iterations, 4 lanes).
const (
	timeCost   uint32 = 3
	memoryCost uint32 = 64 * 1024 // KiB
	threads    uint8  = 4
	saltLength        = 16
	keyLength  uint32 = 32
)

var ErrMalformedHash = errors.New("password: malformed hash encoding")

// Hash derives an Argon2id hash and encodes it in PHC format:
// $argon2id$v=19$m=65536,t=3,p=4$<salt-b64>$<key-b64>
func Hash(plain string) (string, error) {
	salt := make([]byte, saltLength)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("password: generate salt: %w", err)
	}

	key := argon2.IDKey([]byte(plain), salt, timeCost, memoryCost, threads, keyLength)

	return fmt.Sprintf(
		"$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version, memoryCost, timeCost, threads,
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(key),
	), nil
}

// Verify re-derives the key using the parameters stored in encoded and
// compares in constant time.
func Verify(plain, encoded string) (bool, error) {
	parts := strings.Split(encoded, "$")
	if len(parts) != 6 || parts[1] != "argon2id" {
		return false, ErrMalformedHash
	}

	var version int
	if _, err := fmt.Sscanf(parts[2], "v=%d", &version); err != nil {
		return false, ErrMalformedHash
	}
	if version != argon2.Version {
		return false, fmt.Errorf("password: unsupported argon2 version %d", version)
	}

	var m uint32
	var t uint32
	var p uint8
	if _, err := fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &m, &t, &p); err != nil {
		return false, ErrMalformedHash
	}

	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return false, ErrMalformedHash
	}
	expected, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil {
		return false, ErrMalformedHash
	}

	derived := argon2.IDKey([]byte(plain), salt, t, m, p, uint32(len(expected)))

	return subtle.ConstantTimeCompare(derived, expected) == 1, nil
}
