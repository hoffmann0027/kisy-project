package users

import (
	"context"
	"errors"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"kisy-backend/pkg/httpjson"
	"kisy-backend/pkg/httpresponse"
)

var usernamePattern = regexp.MustCompile(`^[A-Za-z0-9_]{3,32}$`)

// maxAvatarUpload bounds the request body for avatar uploads (validation
// happens in the avatars service; this is a coarse transport guard).
const maxAvatarUpload = 1 << 20 // 1 MiB

// AvatarStore stores a validated avatar image and returns its versioned URL.
// Satisfied by *avatars.Service; a local interface avoids a users→avatars
// dependency leaking into the domain.
type AvatarStore interface {
	Store(ctx context.Context, ownerType string, ownerID uuid.UUID, raw []byte) (string, error)
}

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
	avatars  AvatarStore
	identity func(*http.Request) (Identity, bool)
	meta     func(*http.Request) ActorMeta
}

func NewHandler(svc *Service, avatars AvatarStore, identity func(*http.Request) (Identity, bool), meta func(*http.Request) ActorMeta) *Handler {
	return &Handler{svc: svc, avatars: avatars, identity: identity, meta: meta}
}

func (h *Handler) Routes(r chi.Router) {
	r.Get("/me", h.getMe)
	r.Patch("/me", h.patchMe)
	r.Post("/me/avatar", h.uploadAvatar)
	r.Get("/directory", h.directory)
}

// directory lists users the actor may start a chat with (same or lower
// clearance), for the new-chat picker.
func (h *Handler) directory(w http.ResponseWriter, r *http.Request) {
	id, ok := h.identity(r)
	if !ok {
		httpresponse.Fail(w, r, http.StatusUnauthorized, httpresponse.ErrAuthInvalidToken, "authentication required")
		return
	}
	query := r.URL.Query().Get("search")
	limit := 0
	if raw := r.URL.Query().Get("limit"); raw != "" {
		limit, _ = strconv.Atoi(raw)
	}

	list, err := h.svc.Directory(r.Context(), id.UserID, id.RoleLevel, query, limit)
	if err != nil {
		httpresponse.Fail(w, r, http.StatusInternalServerError, httpresponse.ErrInternal, "failed to search users")
		return
	}
	if list == nil {
		list = []DTO{}
	}
	httpresponse.OK(w, r, http.StatusOK, map[string]any{"users": list})
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
	Username    *string `json:"username"`
	DisplayName *string `json:"displayName"`
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
	if req.Username == nil && req.DisplayName == nil {
		httpresponse.Fail(w, r, http.StatusBadRequest, httpresponse.ErrValidationFailed, "no updatable fields provided (supported: username, displayName; password changes go to POST /auth/password)")
		return
	}

	var u *User

	if req.DisplayName != nil {
		name := strings.TrimSpace(*req.DisplayName)
		if n := len([]rune(name)); n < 1 || n > 64 {
			httpresponse.Fail(w, r, http.StatusBadRequest, httpresponse.ErrValidationFailed, "display name must be 1-64 characters")
			return
		}
		updated, err := h.svc.ChangeDisplayName(r.Context(), id.UserID, name, h.meta(r))
		if err != nil {
			httpresponse.Fail(w, r, http.StatusInternalServerError, httpresponse.ErrInternal, "internal error")
			return
		}
		u = updated
	}

	if req.Username != nil {
		if !usernamePattern.MatchString(*req.Username) {
			httpresponse.Fail(w, r, http.StatusBadRequest, httpresponse.ErrValidationFailed, "username must be 3-32 characters: letters, digits, underscore")
			return
		}
		updated, err := h.svc.ChangeUsername(r.Context(), id.UserID, *req.Username, h.meta(r))
		if errors.Is(err, ErrUsernameTaken) {
			httpresponse.Fail(w, r, http.StatusConflict, httpresponse.ErrValidationFailed, "username is already taken")
			return
		}
		if err != nil {
			httpresponse.Fail(w, r, http.StatusInternalServerError, httpresponse.ErrInternal, "internal error")
			return
		}
		u = updated
	}

	httpresponse.OK(w, r, http.StatusOK, map[string]any{"user": u.ToDTO()})
}

// uploadAvatar accepts a raw image body (image/jpeg or image/png), stores it,
// points the user's avatar_url at it and returns the updated profile.
func (h *Handler) uploadAvatar(w http.ResponseWriter, r *http.Request) {
	id, ok := h.identity(r)
	if !ok {
		httpresponse.Fail(w, r, http.StatusUnauthorized, httpresponse.ErrAuthInvalidToken, "authentication required")
		return
	}

	raw, err := io.ReadAll(http.MaxBytesReader(w, r.Body, maxAvatarUpload))
	if err != nil {
		httpresponse.Fail(w, r, http.StatusRequestEntityTooLarge, httpresponse.ErrValidationFailed, "image too large")
		return
	}

	url, err := h.avatars.Store(r.Context(), "user", id.UserID, raw)
	if err != nil {
		httpresponse.Fail(w, r, http.StatusBadRequest, httpresponse.ErrValidationFailed, "invalid image: "+err.Error())
		return
	}

	u, err := h.svc.SetAvatarURL(r.Context(), id.UserID, url)
	if err != nil {
		httpresponse.Fail(w, r, http.StatusInternalServerError, httpresponse.ErrInternal, "internal error")
		return
	}

	httpresponse.OK(w, r, http.StatusOK, map[string]any{"user": u.ToDTO()})
}
