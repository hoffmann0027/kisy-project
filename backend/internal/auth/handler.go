package auth

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net/http"
	"regexp"
	"strings"
	"time"
	"unicode"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"

	"kisy-backend/internal/platform/clientip"
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
	// nativeOrigins are the WebView origins of the packaged apps; only
	// requests from them may receive tokens in a response body (audit A-06).
	nativeOrigins []string
}

// SetNativeOrigins sets the app origins allowed to receive tokens in the
// response body (NATIVE_APP_ORIGINS). Empty: no request ever does.
func (h *Handler) SetNativeOrigins(origins []string) { h.nativeOrigins = origins }

func NewHandler(svc *Service, mw *Middleware, ipHashSalt string, secureCookie bool) *Handler {
	return &Handler{svc: svc, mw: mw, ipHashSalt: ipHashSalt, secureCookie: secureCookie}
}

// Routes mounts the auth endpoints on r. Rate limiting is applied by the
// caller (main router) so limits are configured in one place.
func (h *Handler) Routes(r chi.Router) {
	r.Post("/register", h.register)
	// Unauthenticated by necessity: it is what the sign-up screen asks before
	// deciding whether to offer registration at all. It reveals one boolean
	// about the deployment's policy and nothing about anyone in it.
	r.Get("/registration", h.registrationPolicy)
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

func clientIP(r *http.Request) string { return clientip.From(r) }

type registerRequest struct {
	InviteToken string `json:"inviteToken"`
	Username    string `json:"username"`
	DisplayName string `json:"displayName"`
	Password    string `json:"password"`
}

// registrationPolicy tells the sign-up screen whether an account can be
// created without an invitation here. Offering a form that will be refused is
// worse than not offering it.
func (h *Handler) registrationPolicy(w http.ResponseWriter, r *http.Request) {
	httpresponse.OK(w, r, http.StatusOK, map[string]any{"open": h.svc.RegistrationOpen()})
}

func (h *Handler) register(w http.ResponseWriter, r *http.Request) {
	var req registerRequest
	if err := httpjson.Decode(w, r, &req); err != nil {
		httpresponse.Fail(w, r, http.StatusBadRequest, httpresponse.ErrValidationFailed, "malformed JSON body")
		return
	}
	// An empty invitation token is now a request for an ordinary account, not
	// a malformed request. A non-empty one that turns out to be invalid still
	// fails, in the service.
	if !usernamePattern.MatchString(req.Username) {
		httpresponse.Fail(w, r, http.StatusBadRequest, httpresponse.ErrValidationFailed, "username must be 3-32 characters: letters, digits, underscore")
		return
	}
	if !validPassword(req.Password) {
		httpresponse.Fail(w, r, http.StatusBadRequest, httpresponse.ErrValidationFailed, "password must be 12-128 characters and contain a letter and a digit")
		return
	}

	res, err := h.svc.Register(r.Context(), req.InviteToken, req.Username, req.DisplayName, req.Password, h.ClientMeta(r))
	if err != nil {
		h.writeAuthError(w, r, err)
		return
	}

	h.setAuthCookies(w, res.Tokens)
	httpresponse.OK(w, r, http.StatusCreated, h.withTokens(r, map[string]any{"user": res.User.ToDTO()}, res.Tokens))
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
	httpresponse.OK(w, r, http.StatusOK, h.withTokens(r, map[string]any{"user": res.User.ToDTO()}, res.Tokens))
}

func (h *Handler) refresh(w http.ResponseWriter, r *http.Request) {
	// Native clients have no cookie jar for us: they send the same
	// "<sessionID>.<secret>" string back in the body. Which of the two the
	// token came from decides whether new tokens may go back in the body.
	fromBody := false
	sessionID, plain, ok := uuid.Nil, "", false
	if h.wantsTokenBody(r) {
		sessionID, plain, ok = refreshFromBody(w, r)
		fromBody = ok
	}
	if !ok {
		sessionID, plain, ok = refreshFromRequest(r)
	}
	if !ok && !fromBody {
		sessionID, plain, ok = refreshFromBody(w, r)
		fromBody = ok
	}
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
	payload := map[string]any{"accessExpiresAt": res.Tokens.AccessExpiresAt}
	// A token that came from the HttpOnly cookie is never handed back
	// readable, whatever the request asks for: that would launder the cookie
	// into a token any script on the page can take (audit A-06).
	if fromBody {
		payload = h.withTokens(r, payload, res.Tokens)
	}
	httpresponse.OK(w, r, http.StatusOK, payload)
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
	case users.IsDisplayNameError(err):
		users.FailDisplayName(w, r, err)
	case errors.Is(err, ErrInvalidCredentials):
		httpresponse.Fail(w, r, http.StatusUnauthorized, httpresponse.ErrAuthInvalidCredentials, "invalid username or password")
	case errors.Is(err, ErrAccountLocked):
		httpresponse.Fail(w, r, http.StatusTooManyRequests, httpresponse.ErrRateLimited, "account temporarily locked, try again later")
	case errors.Is(err, ErrInvalidInvite):
		httpresponse.Fail(w, r, http.StatusBadRequest, httpresponse.ErrAuthInvalidToken, "invitation token is invalid or expired")
	case errors.Is(err, ErrRegistrationClosed):
		httpresponse.Fail(w, r, http.StatusForbidden, httpresponse.ErrAccessDenied, "registration requires an invitation on this deployment")
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

// refreshFromBody reads the refresh token a native client sends in the body.
func refreshFromBody(w http.ResponseWriter, r *http.Request) (uuid.UUID, string, bool) {
	var body struct {
		RefreshToken string `json:"refreshToken"`
	}
	if err := httpjson.Decode(w, r, &body); err != nil {
		return uuid.Nil, "", false
	}
	return splitRefreshToken(body.RefreshToken)
}

// refreshFromRequest parses the "<sessionID>.<token>" refresh cookie.
func refreshFromRequest(r *http.Request) (uuid.UUID, string, bool) {
	c, err := r.Cookie(RefreshCookieName)
	if err != nil || c.Value == "" {
		return uuid.Nil, "", false
	}
	return splitRefreshToken(c.Value)
}

// --- native (mobile app) clients ---
//
// The Android/iOS shell runs the same SPA inside a WebView, but on its own
// origin (https://localhost), so the API is cross-site for it and the
// HttpOnly+SameSite=Strict cookies below are never sent. Such clients
// authenticate with `Authorization: Bearer` instead — the middleware already
// accepts that — which means they need the raw tokens in the response body.
//
// The header alone used to be enough, and any script can set a header: a XSS
// on the web app sent it to /auth/refresh, the browser attached the HttpOnly
// refresh cookie by itself, and the response carried a readable 30-day token
// (audit A-06). Two conditions now, neither of which a page on the web origin
// can meet:
//   - the Origin is one of NATIVE_APP_ORIGINS. Browsers set Origin and scripts
//     cannot change it; the app's WebView sends https://localhost, which the
//     CSRF check already requires of it;
//   - on refresh, the token came from the body, never from the cookie.
//
// Installed apps already satisfy both, so nothing on the phones changes.
const nativeClientHeader = "X-Kisy-Client"

// wantsTokenBody reports whether the caller asked for tokens in the payload
// and is an app origin allowed to get them.
func (h *Handler) wantsTokenBody(r *http.Request) bool {
	if !strings.EqualFold(r.Header.Get(nativeClientHeader), "native") {
		return false
	}
	origin := strings.TrimRight(r.Header.Get("Origin"), "/")
	if origin == "" {
		return false
	}
	for _, allowed := range h.nativeOrigins {
		if strings.EqualFold(origin, strings.TrimRight(allowed, "/")) {
			return true
		}
	}
	return false
}

// tokenBody is the token envelope handed to native clients.
type tokenBody struct {
	AccessToken     string    `json:"accessToken"`
	AccessExpiresAt time.Time `json:"accessExpiresAt"`
	// RefreshToken is the same "<sessionID>.<secret>" string the refresh
	// cookie carries; native clients send it back in the refresh request body.
	RefreshToken   string    `json:"refreshToken"`
	RefreshExpires time.Time `json:"refreshExpires"`
}

func newTokenBody(t TokenPair) tokenBody {
	return tokenBody{
		AccessToken:     t.AccessToken,
		AccessExpiresAt: t.AccessExpiresAt,
		RefreshToken:    t.RefreshCookie,
		RefreshExpires:  t.RefreshExpires,
	}
}

// withTokens adds the token envelope to a response payload when the caller is
// a native client; browsers get the payload untouched.
func (h *Handler) withTokens(r *http.Request, payload map[string]any, t TokenPair) map[string]any {
	if h.wantsTokenBody(r) {
		payload["tokens"] = newTokenBody(t)
	}
	return payload
}

// splitRefreshToken parses the "<sessionID>.<secret>" refresh token, shared by
// the cookie and the native request-body paths.
func splitRefreshToken(v string) (uuid.UUID, string, bool) {
	sidRaw, plain, found := strings.Cut(v, ".")
	if !found || plain == "" {
		return uuid.Nil, "", false
	}
	sid, err := uuid.Parse(sidRaw)
	if err != nil {
		return uuid.Nil, "", false
	}
	return sid, plain, true
}
