package auth

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net"
	"net/http"
	"regexp"
	"strings"
	"unicode"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"

	"kisy-backend/internal/users"
	"kisy-backend/pkg/httpjson"
	"kisy-backend/pkg/httpresponse"
)

// Cookie names. The refresh cookie is path-scoped to the auth endpoints so
// it is never sent with ordinary API traffic.
const (
	AccessCookieName  = "kisy_access"
	RefreshCookieName = "kisy_refresh"
	refreshCookiePath = "/api/v1/auth"
)

var usernamePattern = regexp.MustCompile(`^[A-Za-z0-9_]{3,32}$`)

// Password policy per docs/spec/06-security.md: length 12-128 with at
// least one letter and one digit.
func validPassword(p string) bool {
	if len(p) < 12 || len(p) > 128 {
		return false
	}
	var hasLetter, hasDigit bool
	for _, r := range p {
		switch {
		case unicode.IsLetter(r):
			hasLetter = true
		case unicode.IsDigit(r):
			hasDigit = true
		}
	}
	return hasLetter && hasDigit
}

// Handler exposes the /auth/* endpoints.
type Handler struct {
	svc          *Service
	mw           *Middleware
	ipHashSalt   string
	secureCookie bool
}

func NewHandler(svc *Service, mw *Middleware, ipHashSalt string, secureCookie bool) *Handler {
	return &Handler{svc: svc, mw: mw, ipHashSalt: ipHashSalt, secureCookie: secureCookie}
}

// Routes mounts the auth endpoints on r. Rate limiting is applied by the
// caller (main router) so limits are configured in one place.
func (h *Handler) Routes(r chi.Router) {
	r.Post("/register", h.register)
	r.Post("/login", h.login)
	r.Post("/refresh", h.refresh)

	r.Group(func(r chi.Router) {
		r.Use(h.mw.RequireAuth)
		r.Post("/logout", h.logout)
		r.Post("/logout-all", h.logoutAll)
		r.Post("/password", h.changePassword)
	})
}

// ClientMeta extracts audit/session attributes from the request.
func (h *Handler) ClientMeta(r *http.Request) ClientMeta {
	return ClientMeta{
		IPHash:     h.HashIP(clientIP(r)),
		UserAgent:  r.UserAgent(),
		DeviceName: r.Header.Get("X-Device-Name"),
		RequestID:  middleware.GetReqID(r.Context()),
	}
}

// HashIP produces the salted digest stored instead of raw addresses.
func (h *Handler) HashIP(ip string) string {
	sum := sha256.Sum256([]byte(h.ipHashSalt + "|" + ip))
	return hex.EncodeToString(sum[:])
}

func clientIP(r *http.Request) string {
	// chi middleware.RealIP has already resolved X-Forwarded-For into
	// RemoteAddr; it may or may not carry a port.
	if host, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
		return host
	}
	return r.RemoteAddr
}

type registerRequest struct {
	InviteToken string `json:"inviteToken"`
	Username    string `json:"username"`
	Password    string `json:"password"`
}

func (h *Handler) register(w http.ResponseWriter, r *http.Request) {
	var req registerRequest
	if err := httpjson.Decode(w, r, &req); err != nil {
		httpresponse.Fail(w, r, http.StatusBadRequest, httpresponse.ErrValidationFailed, "malformed JSON body")
		return
	}
	if req.InviteToken == "" {
		httpresponse.Fail(w, r, http.StatusBadRequest, httpresponse.ErrValidationFailed, "inviteToken is required")
		return
	}
	if !usernamePattern.MatchString(req.Username) {
		httpresponse.Fail(w, r, http.StatusBadRequest, httpresponse.ErrValidationFailed, "username must be 3-32 characters: letters, digits, underscore")
		return
	}
	if !validPassword(req.Password) {
		httpresponse.Fail(w, r, http.StatusBadRequest, httpresponse.ErrValidationFailed, "password must be 12-128 characters and contain a letter and a digit")
		return
	}

	res, err := h.svc.Register(r.Context(), req.InviteToken, req.Username, req.Password, h.ClientMeta(r))
	if err != nil {
		h.writeAuthError(w, r, err)
		return
	}

	h.setAuthCookies(w, res.Tokens)
	httpresponse.OK(w, r, http.StatusCreated, map[string]any{"user": res.User.ToDTO()})
}

type loginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func (h *Handler) login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := httpjson.Decode(w, r, &req); err != nil {
		httpresponse.Fail(w, r, http.StatusBadRequest, httpresponse.ErrValidationFailed, "malformed JSON body")
		return
	}
	if req.Username == "" || req.Password == "" {
		httpresponse.Fail(w, r, http.StatusBadRequest, httpresponse.ErrValidationFailed, "username and password are required")
		return
	}

	res, err := h.svc.Login(r.Context(), req.Username, req.Password, h.ClientMeta(r))
	if err != nil {
		h.writeAuthError(w, r, err)
		return
	}

	h.setAuthCookies(w, res.Tokens)
	httpresponse.OK(w, r, http.StatusOK, map[string]any{"user": res.User.ToDTO()})
}

func (h *Handler) refresh(w http.ResponseWriter, r *http.Request) {
	sessionID, plain, ok := refreshFromRequest(r)
	if !ok {
		httpresponse.Fail(w, r, http.StatusUnauthorized, httpresponse.ErrAuthInvalidToken, "missing refresh token")
		return
	}

	res, err := h.svc.Refresh(r.Context(), sessionID, plain, h.ClientMeta(r))
	if err != nil {
		h.clearAuthCookies(w)
		h.writeAuthError(w, r, err)
		return
	}

	h.setAuthCookies(w, res.Tokens)
	httpresponse.OK(w, r, http.StatusOK, map[string]any{
		"accessExpiresAt": res.Tokens.AccessExpiresAt,
	})
}

func (h *Handler) logout(w http.ResponseWriter, r *http.Request) {
	claims, _ := ClaimsFromContext(r.Context())

	if err := h.svc.Logout(r.Context(), claims.UserID, claims.SessionID, h.ClientMeta(r)); err != nil {
		httpresponse.Fail(w, r, http.StatusInternalServerError, httpresponse.ErrInternal, "logout failed")
		return
	}

	h.clearAuthCookies(w)
	httpresponse.OK(w, r, http.StatusOK, map[string]any{"loggedOut": true})
}

func (h *Handler) logoutAll(w http.ResponseWriter, r *http.Request) {
	claims, _ := ClaimsFromContext(r.Context())

	n, err := h.svc.LogoutAll(r.Context(), claims.UserID, claims.SessionID, h.ClientMeta(r))
	if err != nil {
		httpresponse.Fail(w, r, http.StatusInternalServerError, httpresponse.ErrInternal, "logout failed")
		return
	}

	h.clearAuthCookies(w)
	httpresponse.OK(w, r, http.StatusOK, map[string]any{"revokedSessions": n})
}

type changePasswordRequest struct {
	CurrentPassword string `json:"currentPassword"`
	NewPassword     string `json:"newPassword"`
}

func (h *Handler) changePassword(w http.ResponseWriter, r *http.Request) {
	claims, _ := ClaimsFromContext(r.Context())

	var req changePasswordRequest
	if err := httpjson.Decode(w, r, &req); err != nil {
		httpresponse.Fail(w, r, http.StatusBadRequest, httpresponse.ErrValidationFailed, "malformed JSON body")
		return
	}
	if req.CurrentPassword == "" {
		httpresponse.Fail(w, r, http.StatusBadRequest, httpresponse.ErrValidationFailed, "currentPassword is required")
		return
	}
	if !validPassword(req.NewPassword) {
		httpresponse.Fail(w, r, http.StatusBadRequest, httpresponse.ErrValidationFailed, "newPassword must be 12-128 characters and contain a letter and a digit")
		return
	}

	err := h.svc.ChangePassword(r.Context(), claims.UserID, claims.SessionID, req.CurrentPassword, req.NewPassword, h.ClientMeta(r))
	if err != nil {
		h.writeAuthError(w, r, err)
		return
	}

	httpresponse.OK(w, r, http.StatusOK, map[string]any{"passwordChanged": true})
}

// writeAuthError maps service errors onto the API error contract without
// leaking internals.
func (h *Handler) writeAuthError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, ErrInvalidCredentials):
		httpresponse.Fail(w, r, http.StatusUnauthorized, httpresponse.ErrAuthInvalidCredentials, "invalid username or password")
	case errors.Is(err, ErrAccountLocked):
		httpresponse.Fail(w, r, http.StatusTooManyRequests, httpresponse.ErrRateLimited, "account temporarily locked, try again later")
	case errors.Is(err, ErrInvalidInvite):
		httpresponse.Fail(w, r, http.StatusBadRequest, httpresponse.ErrAuthInvalidToken, "invitation token is invalid or expired")
	case errors.Is(err, users.ErrUsernameTaken):
		httpresponse.Fail(w, r, http.StatusConflict, httpresponse.ErrValidationFailed, "username is already taken")
	case errors.Is(err, ErrInvalidRefresh):
		httpresponse.Fail(w, r, http.StatusUnauthorized, httpresponse.ErrAuthInvalidToken, "refresh token is invalid")
	default:
		httpresponse.Fail(w, r, http.StatusInternalServerError, httpresponse.ErrInternal, "internal error")
	}
}

// Cookies below are HttpOnly + SameSite=Strict; Secure is h.secureCookie,
// which is literal true whenever APP_ENV=production (dev serves plain http).
// gosec cannot see through the variable, hence the G124 annotations.
func (h *Handler) setAuthCookies(w http.ResponseWriter, t TokenPair) {
	http.SetCookie(w, &http.Cookie{ // #nosec G124 -- Secure=true in production
		Name:     AccessCookieName,
		Value:    t.AccessToken,
		Path:     "/",
		Expires:  t.AccessExpiresAt,
		HttpOnly: true,
		Secure:   h.secureCookie,
		SameSite: http.SameSiteStrictMode,
	})
	http.SetCookie(w, &http.Cookie{ // #nosec G124 -- Secure=true in production
		Name:     RefreshCookieName,
		Value:    t.RefreshCookie,
		Path:     refreshCookiePath,
		Expires:  t.RefreshExpires,
		HttpOnly: true,
		Secure:   h.secureCookie,
		SameSite: http.SameSiteStrictMode,
	})
}

func (h *Handler) clearAuthCookies(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{ // #nosec G124 -- expiring cookie, Secure=true in production
		Name: AccessCookieName, Value: "", Path: "/", MaxAge: -1,
		HttpOnly: true, Secure: h.secureCookie, SameSite: http.SameSiteStrictMode,
	})
	http.SetCookie(w, &http.Cookie{ // #nosec G124 -- expiring cookie, Secure=true in production
		Name: RefreshCookieName, Value: "", Path: refreshCookiePath, MaxAge: -1,
		HttpOnly: true, Secure: h.secureCookie, SameSite: http.SameSiteStrictMode,
	})
}

// refreshFromRequest parses the "<sessionID>.<token>" refresh cookie.
func refreshFromRequest(r *http.Request) (uuid.UUID, string, bool) {
	c, err := r.Cookie(RefreshCookieName)
	if err != nil || c.Value == "" {
		return uuid.Nil, "", false
	}
	sidRaw, plain, found := strings.Cut(c.Value, ".")
	if !found || plain == "" {
		return uuid.Nil, "", false
	}
	sid, err := uuid.Parse(sidRaw)
	if err != nil {
		return uuid.Nil, "", false
	}
	return sid, plain, true
}
