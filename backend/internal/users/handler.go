package users

import (
	"errors"
	"net/http"
	"regexp"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"kisy-backend/pkg/httpjson"
	"kisy-backend/pkg/httpresponse"
)

var usernamePattern = regexp.MustCompile(`^[A-Za-z0-9_]{3,32}$`)

// Identity is the authenticated actor view this module needs. It is
// supplied by the main router as a closure over the auth middleware,
// because auth imports users and a direct users→auth import would cycle.
type Identity struct {
	UserID    uuid.UUID
	SessionID uuid.UUID
	RoleLevel int
}

// Handler exposes /users/* endpoints. RequireAuth is applied by the main
// router before these routes.
type Handler struct {
	svc      *Service
	identity func(*http.Request) (Identity, bool)
	meta     func(*http.Request) ActorMeta
}

func NewHandler(svc *Service, identity func(*http.Request) (Identity, bool), meta func(*http.Request) ActorMeta) *Handler {
	return &Handler{svc: svc, identity: identity, meta: meta}
}

func (h *Handler) Routes(r chi.Router) {
	r.Get("/me", h.getMe)
	r.Patch("/me", h.patchMe)
}

func (h *Handler) getMe(w http.ResponseWriter, r *http.Request) {
	id, ok := h.identity(r)
	if !ok {
		httpresponse.Fail(w, r, http.StatusUnauthorized, httpresponse.ErrAuthInvalidToken, "authentication required")
		return
	}

	u, err := h.svc.GetByID(r.Context(), id.UserID)
	if errors.Is(err, ErrNotFound) {
		httpresponse.Fail(w, r, http.StatusNotFound, httpresponse.ErrResourceNotFound, "user not found")
		return
	}
	if err != nil {
		httpresponse.Fail(w, r, http.StatusInternalServerError, httpresponse.ErrInternal, "internal error")
		return
	}

	httpresponse.OK(w, r, http.StatusOK, map[string]any{"user": u.ToDTO()})
}

type patchMeRequest struct {
	Username *string `json:"username"`
}

func (h *Handler) patchMe(w http.ResponseWriter, r *http.Request) {
	id, ok := h.identity(r)
	if !ok {
		httpresponse.Fail(w, r, http.StatusUnauthorized, httpresponse.ErrAuthInvalidToken, "authentication required")
		return
	}

	var req patchMeRequest
	if err := httpjson.Decode(w, r, &req); err != nil {
		httpresponse.Fail(w, r, http.StatusBadRequest, httpresponse.ErrValidationFailed, "malformed JSON body")
		return
	}
	if req.Username == nil {
		httpresponse.Fail(w, r, http.StatusBadRequest, httpresponse.ErrValidationFailed, "no updatable fields provided (supported: username; password changes go to POST /auth/password)")
		return
	}
	if !usernamePattern.MatchString(*req.Username) {
		httpresponse.Fail(w, r, http.StatusBadRequest, httpresponse.ErrValidationFailed, "username must be 3-32 characters: letters, digits, underscore")
		return
	}

	u, err := h.svc.ChangeUsername(r.Context(), id.UserID, *req.Username, h.meta(r))
	if errors.Is(err, ErrUsernameTaken) {
		httpresponse.Fail(w, r, http.StatusConflict, httpresponse.ErrValidationFailed, "username is already taken")
		return
	}
	if err != nil {
		httpresponse.Fail(w, r, http.StatusInternalServerError, httpresponse.ErrInternal, "internal error")
		return
	}

	httpresponse.OK(w, r, http.StatusOK, map[string]any{"user": u.ToDTO()})
}
