package auth

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"kisy-backend/internal/auth/token"
	"kisy-backend/pkg/httpresponse"
)

type ctxKey struct{}

// ClaimsFromContext returns the authenticated claims placed by RequireAuth.
func ClaimsFromContext(ctx context.Context) (*token.AccessClaims, bool) {
	claims, ok := ctx.Value(ctxKey{}).(*token.AccessClaims)
	return claims, ok
}

// Middleware authenticates requests and enforces clearance levels.
type Middleware struct {
	tokens   *token.Manager
	sessions SessionRepository
	pool     *pgxpool.Pool
}

func NewMiddleware(tokens *token.Manager, sessions SessionRepository, pool *pgxpool.Pool) *Middleware {
	return &Middleware{tokens: tokens, sessions: sessions, pool: pool}
}

// RequireAuth validates the access token (cookie or Bearer header) and
// verifies the backing session is still live, so logout and password
// changes take effect immediately rather than at access-token expiry.
func (m *Middleware) RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw := accessTokenFromRequest(r)
		if raw == "" {
			httpresponse.Fail(w, r, http.StatusUnauthorized, httpresponse.ErrAuthInvalidToken, "authentication required")
			return
		}

		claims, err := m.tokens.ParseAccess(raw)
		if errors.Is(err, token.ErrExpired) {
			httpresponse.Fail(w, r, http.StatusUnauthorized, httpresponse.ErrAuthExpired, "access token expired")
			return
		}
		if err != nil {
			httpresponse.Fail(w, r, http.StatusUnauthorized, httpresponse.ErrAuthInvalidToken, "invalid access token")
			return
		}

		sess, err := m.sessions.GetByID(r.Context(), m.pool, claims.SessionID)
		if err != nil || !sess.Active(time.Now().UTC()) || sess.UserID != claims.UserID {
			httpresponse.Fail(w, r, http.StatusUnauthorized, httpresponse.ErrAuthInvalidToken, "session is no longer active")
			return
		}

		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), ctxKey{}, claims)))
	})
}

// Authenticate validates a request's access token (cookie, Bearer header,
// or access_token query parameter) and confirms the session is live. It is
// used by non-middleware entry points such as the WebSocket upgrade, where
// the token may arrive as a query parameter because browsers cannot set
// headers on a WebSocket handshake. Returns nil when authentication fails.
func (m *Middleware) Authenticate(r *http.Request) *token.AccessClaims {
	raw := accessTokenFromRequest(r)
	if raw == "" {
		raw = r.URL.Query().Get("access_token")
	}
	if raw == "" {
		return nil
	}

	claims, err := m.tokens.ParseAccess(raw)
	if err != nil {
		return nil
	}

	sess, err := m.sessions.GetByID(r.Context(), m.pool, claims.SessionID)
	if err != nil || !sess.Active(time.Now().UTC()) || sess.UserID != claims.UserID {
		return nil
	}
	return claims
}

// RequireClearance allows only actors whose level is at most maxLevel
// (numerically lower level = higher privilege; 1 = CEO).
func (m *Middleware) RequireClearance(maxLevel int) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims, ok := ClaimsFromContext(r.Context())
			if !ok {
				httpresponse.Fail(w, r, http.StatusUnauthorized, httpresponse.ErrAuthInvalidToken, "authentication required")
				return
			}
			if claims.RoleLevel > maxLevel {
				httpresponse.Fail(w, r, http.StatusForbidden, httpresponse.ErrAccessDenied, "insufficient clearance")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func accessTokenFromRequest(r *http.Request) string {
	if c, err := r.Cookie(AccessCookieName); err == nil && c.Value != "" {
		return c.Value
	}
	if h := r.Header.Get("Authorization"); strings.HasPrefix(h, "Bearer ") {
		return strings.TrimPrefix(h, "Bearer ")
	}
	return ""
}
