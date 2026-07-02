// Package token issues and validates the two token kinds used by KISY:
// short-lived JWT access tokens and opaque, hashed refresh tokens. The
// same opaque-token helper also backs invitation tokens.
package token

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

var (
	ErrExpired = errors.New("token: expired")
	ErrInvalid = errors.New("token: invalid")
)

const issuer = "kisy"

// AccessClaims is the validated content of an access token.
type AccessClaims struct {
	UserID    uuid.UUID
	SessionID uuid.UUID
	RoleLevel int
}

type jwtClaims struct {
	SessionID string `json:"sid"`
	RoleLevel int    `json:"lvl"`
	jwt.RegisteredClaims
}

// Manager signs and parses access tokens (HS256).
type Manager struct {
	secret []byte
	ttl    time.Duration
}

func NewManager(secret string, ttl time.Duration) *Manager {
	return &Manager{secret: []byte(secret), ttl: ttl}
}

func (m *Manager) IssueAccess(userID, sessionID uuid.UUID, roleLevel int) (string, time.Time, error) {
	now := time.Now().UTC()
	expiresAt := now.Add(m.ttl)

	claims := jwtClaims{
		SessionID: sessionID.String(),
		RoleLevel: roleLevel,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    issuer,
			Subject:   userID.String(),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(expiresAt),
		},
	}

	signed, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(m.secret)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("token: sign access token: %w", err)
	}
	return signed, expiresAt, nil
}

func (m *Manager) ParseAccess(raw string) (*AccessClaims, error) {
	var claims jwtClaims

	_, err := jwt.ParseWithClaims(raw, &claims,
		func(t *jwt.Token) (any, error) { return m.secret, nil },
		jwt.WithValidMethods([]string{"HS256"}),
		jwt.WithIssuer(issuer),
		jwt.WithExpirationRequired(),
	)
	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrExpired
		}
		return nil, ErrInvalid
	}

	userID, err := uuid.Parse(claims.Subject)
	if err != nil {
		return nil, ErrInvalid
	}
	sessionID, err := uuid.Parse(claims.SessionID)
	if err != nil {
		return nil, ErrInvalid
	}
	if claims.RoleLevel < 1 || claims.RoleLevel > 10 {
		return nil, ErrInvalid
	}

	return &AccessClaims{UserID: userID, SessionID: sessionID, RoleLevel: claims.RoleLevel}, nil
}

// NewOpaqueToken returns a 256-bit cryptographically random token and its
// SHA-256 hex digest. Only the digest is ever persisted; the plaintext is
// shown to the client once. Used for refresh and invitation tokens.
func NewOpaqueToken() (plain, digest string, err error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", "", fmt.Errorf("token: generate opaque token: %w", err)
	}
	plain = base64.RawURLEncoding.EncodeToString(buf)
	return plain, HashOpaqueToken(plain), nil
}

// HashOpaqueToken computes the storage digest of an opaque token.
func HashOpaqueToken(plain string) string {
	sum := sha256.Sum256([]byte(plain))
	return hex.EncodeToString(sum[:])
}
