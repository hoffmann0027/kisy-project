package admin

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"kisy-backend/internal/audit"
	"kisy-backend/pkg/httpjson"
	"kisy-backend/pkg/httpresponse"
)

// Handler exposes /admin. The router gates every route behind
// RequireAuth + RequireClearance(1).
type Handler struct {
	svc   *Service
	audit *audit.Reader
	actor func(*http.Request) (ActorMeta, bool)
}

func NewHandler(svc *Service, auditReader *audit.Reader, actor func(*http.Request) (ActorMeta, bool)) *Handler {
	return &Handler{svc: svc, audit: auditReader, actor: actor}
}

func (h *Handler) Routes(r chi.Router) {
	r.Get("/users", h.listUsers)
	r.Patch("/users/{userID}/role", h.changeRole)
	r.Post("/users/{userID}/reset-password", h.resetPassword)
	r.Post("/users/{userID}/activate", h.activate)
	r.Post("/users/{userID}/deactivate", h.deactivate)
	r.Get("/audit", h.auditLog)
}

func (h *Handler) listUsers(w http.ResponseWriter, r *http.Request) {
	if _, ok := h.actor(r); !ok {
		unauth(w, r)
		return
	}
	limit, offset := pageParams(r)
	list, err := h.svc.ListUsers(r.Context(), limit, offset)
	if err != nil {
		internal(w, r)
		return
	}
	httpresponse.OK(w, r, http.StatusOK, map[string]any{"users": list})
}

type changeRoleRequest struct {
	RoleLevel int `json:"roleLevel"`
}

func (h *Handler) changeRole(w http.ResponseWriter, r *http.Request) {
	actor, ok := h.actor(r)
	if !ok {
		unauth(w, r)
		return
	}
	targetID, ok := parseTarget(w, r)
	if !ok {
		return
	}
	var req changeRoleRequest
	if err := httpjson.Decode(w, r, &req); err != nil {
		httpresponse.Fail(w, r, http.StatusBadRequest, httpresponse.ErrValidationFailed, "malformed JSON body")
		return
	}
	h.writeResult(w, r, h.svc.ChangeRole(r.Context(), targetID, req.RoleLevel, actor))
}

type resetPasswordRequest struct {
	NewPassword string `json:"newPassword"`
}

func (h *Handler) resetPassword(w http.ResponseWriter, r *http.Request) {
	actor, ok := h.actor(r)
	if !ok {
		unauth(w, r)
		return
	}
	targetID, ok := parseTarget(w, r)
	if !ok {
		return
	}
	var req resetPasswordRequest
	if err := httpjson.Decode(w, r, &req); err != nil {
		httpresponse.Fail(w, r, http.StatusBadRequest, httpresponse.ErrValidationFailed, "malformed JSON body")
		return
	}
	h.writeResult(w, r, h.svc.ResetPassword(r.Context(), targetID, req.NewPassword, actor))
}

func (h *Handler) activate(w http.ResponseWriter, r *http.Request)   { h.setActive(w, r, true) }
func (h *Handler) deactivate(w http.ResponseWriter, r *http.Request) { h.setActive(w, r, false) }

func (h *Handler) setActive(w http.ResponseWriter, r *http.Request, active bool) {
	actor, ok := h.actor(r)
	if !ok {
		unauth(w, r)
		return
	}
	targetID, ok := parseTarget(w, r)
	if !ok {
		return
	}
	h.writeResult(w, r, h.svc.SetActive(r.Context(), targetID, active, actor))
}

func (h *Handler) auditLog(w http.ResponseWriter, r *http.Request) {
	if _, ok := h.actor(r); !ok {
		unauth(w, r)
		return
	}
	limit, offset := pageParams(r)
	entries, err := h.audit.List(r.Context(), r.URL.Query().Get("action"), limit, offset)
	if err != nil {
		internal(w, r)
		return
	}
	if entries == nil {
		entries = []audit.LogEntry{}
	}
	httpresponse.OK(w, r, http.StatusOK, map[string]any{"entries": entries})
}

func (h *Handler) writeResult(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case err == nil:
		httpresponse.OK(w, r, http.StatusOK, map[string]any{"ok": true})
	case errors.Is(err, ErrInvalidRole), errors.Is(err, ErrWeakPassword):
		httpresponse.Fail(w, r, http.StatusBadRequest, httpresponse.ErrValidationFailed, err.Error())
	case errors.Is(err, ErrSelfMutation):
		httpresponse.Fail(w, r, http.StatusForbidden, httpresponse.ErrAccessDenied, err.Error())
	case errors.Is(err, ErrNotFound):
		httpresponse.Fail(w, r, http.StatusNotFound, httpresponse.ErrResourceNotFound, "user not found")
	default:
		internal(w, r)
	}
}

func parseTarget(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	id, err := uuid.Parse(chi.URLParam(r, "userID"))
	if err != nil {
		httpresponse.Fail(w, r, http.StatusNotFound, httpresponse.ErrResourceNotFound, "user not found")
		return uuid.Nil, false
	}
	return id, true
}

func pageParams(r *http.Request) (limit, offset int) {
	limit = 50
	if v, err := strconv.Atoi(r.URL.Query().Get("limit")); err == nil {
		limit = v
	}
	if v, err := strconv.Atoi(r.URL.Query().Get("offset")); err == nil {
		offset = v
	}
	return limit, offset
}

func unauth(w http.ResponseWriter, r *http.Request) {
	httpresponse.Fail(w, r, http.StatusUnauthorized, httpresponse.ErrAuthInvalidToken, "authentication required")
}

func internal(w http.ResponseWriter, r *http.Request) {
	httpresponse.Fail(w, r, http.StatusInternalServerError, httpresponse.ErrInternal, "internal error")
}
