package groups

import (
	"context"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"kisy-backend/pkg/httpjson"
	"kisy-backend/pkg/httpresponse"
)

// UserLookup resolves a target user's clearance level, injected by the
// main router to avoid a groups→users import cycle. It returns false when
// the user does not exist.
type UserLookup func(ctx context.Context, id uuid.UUID) (level int, ok bool)

// Handler exposes /groups endpoints. RequireAuth is applied by the router.
type Handler struct {
	svc    *Service
	actor  func(*http.Request) (ActorMeta, bool)
	lookup UserLookup
}

func NewHandler(svc *Service, actor func(*http.Request) (ActorMeta, bool), lookup UserLookup) *Handler {
	return &Handler{svc: svc, actor: actor, lookup: lookup}
}

func (h *Handler) Routes(r chi.Router) {
	r.Get("/", h.list)
	r.Post("/", h.create) // any user may create (clearance validated in service)
	r.Get("/{groupID}", h.get)
	r.Delete("/{groupID}", h.delete)
	r.Get("/{groupID}/members", h.listMembers)
	r.Post("/{groupID}/members", h.addMember)
}

func (h *Handler) delete(w http.ResponseWriter, r *http.Request) {
	actor, ok := h.actor(r)
	if !ok {
		httpresponse.Fail(w, r, http.StatusUnauthorized, httpresponse.ErrAuthInvalidToken, "authentication required")
		return
	}
	id, err := uuid.Parse(chi.URLParam(r, "groupID"))
	if err != nil {
		httpresponse.Fail(w, r, http.StatusNotFound, httpresponse.ErrResourceNotFound, "group not found")
		return
	}

	err = h.svc.Delete(r.Context(), id, actor)
	switch {
	case errors.Is(err, ErrNotFound):
		httpresponse.Fail(w, r, http.StatusNotFound, httpresponse.ErrResourceNotFound, "group not found")
	case errors.Is(err, ErrForbidden):
		httpresponse.Fail(w, r, http.StatusForbidden, httpresponse.ErrAccessDenied, "only the CEO or the group founder may delete this group")
	case err != nil:
		httpresponse.Fail(w, r, http.StatusInternalServerError, httpresponse.ErrInternal, "failed to delete group")
	default:
		httpresponse.OK(w, r, http.StatusOK, map[string]any{"deleted": true})
	}
}

func (h *Handler) listMembers(w http.ResponseWriter, r *http.Request) {
	actor, ok := h.actor(r)
	if !ok {
		httpresponse.Fail(w, r, http.StatusUnauthorized, httpresponse.ErrAuthInvalidToken, "authentication required")
		return
	}
	id, err := uuid.Parse(chi.URLParam(r, "groupID"))
	if err != nil {
		httpresponse.Fail(w, r, http.StatusNotFound, httpresponse.ErrResourceNotFound, "group not found")
		return
	}
	members, err := h.svc.ListMembers(r.Context(), id, actor)
	if errors.Is(err, ErrNotFound) {
		httpresponse.Fail(w, r, http.StatusNotFound, httpresponse.ErrResourceNotFound, "group not found")
		return
	}
	if err != nil {
		httpresponse.Fail(w, r, http.StatusInternalServerError, httpresponse.ErrInternal, "internal error")
		return
	}
	httpresponse.OK(w, r, http.StatusOK, map[string]any{"members": members})
}

type createRequest struct {
	Name         string  `json:"name"`
	Description  *string `json:"description"`
	MinRoleLevel int     `json:"minRoleLevel"`
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	actor, ok := h.actor(r)
	if !ok {
		httpresponse.Fail(w, r, http.StatusUnauthorized, httpresponse.ErrAuthInvalidToken, "authentication required")
		return
	}

	var req createRequest
	if err := httpjson.Decode(w, r, &req); err != nil {
		httpresponse.Fail(w, r, http.StatusBadRequest, httpresponse.ErrValidationFailed, "malformed JSON body")
		return
	}
	if l := len(req.Name); l < 1 || l > 128 {
		httpresponse.Fail(w, r, http.StatusBadRequest, httpresponse.ErrValidationFailed, "name must be 1-128 characters")
		return
	}
	if req.MinRoleLevel < 1 || req.MinRoleLevel > 10 {
		httpresponse.Fail(w, r, http.StatusBadRequest, httpresponse.ErrValidationFailed, "minRoleLevel must be between 1 and 10")
		return
	}

	g, err := h.svc.Create(r.Context(), CreateInput{
		Name:         req.Name,
		Description:  req.Description,
		MinRoleLevel: req.MinRoleLevel,
	}, actor)
	if errors.Is(err, ErrLevelTooHigh) {
		httpresponse.Fail(w, r, http.StatusForbidden, httpresponse.ErrAccessDenied, "нельзя создать группу с уровнем доступа выше вашего")
		return
	}
	if err != nil {
		httpresponse.Fail(w, r, http.StatusInternalServerError, httpresponse.ErrInternal, "failed to create group")
		return
	}

	httpresponse.OK(w, r, http.StatusCreated, map[string]any{"group": g.ToDTO()})
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	actor, ok := h.actor(r)
	if !ok {
		httpresponse.Fail(w, r, http.StatusUnauthorized, httpresponse.ErrAuthInvalidToken, "authentication required")
		return
	}

	list, err := h.svc.ListVisible(r.Context(), actor)
	if err != nil {
		httpresponse.Fail(w, r, http.StatusInternalServerError, httpresponse.ErrInternal, "failed to list groups")
		return
	}

	dtos := make([]DTO, 0, len(list))
	for i := range list {
		dtos = append(dtos, list[i].ToDTO())
	}
	httpresponse.OK(w, r, http.StatusOK, map[string]any{"groups": dtos})
}

func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	actor, ok := h.actor(r)
	if !ok {
		httpresponse.Fail(w, r, http.StatusUnauthorized, httpresponse.ErrAuthInvalidToken, "authentication required")
		return
	}
	id, err := uuid.Parse(chi.URLParam(r, "groupID"))
	if err != nil {
		httpresponse.Fail(w, r, http.StatusNotFound, httpresponse.ErrResourceNotFound, "group not found")
		return
	}

	g, err := h.svc.Get(r.Context(), id, actor)
	if errors.Is(err, ErrNotFound) {
		httpresponse.Fail(w, r, http.StatusNotFound, httpresponse.ErrResourceNotFound, "group not found")
		return
	}
	if err != nil {
		httpresponse.Fail(w, r, http.StatusInternalServerError, httpresponse.ErrInternal, "internal error")
		return
	}

	httpresponse.OK(w, r, http.StatusOK, map[string]any{"group": g.ToDTO()})
}

type addMemberRequest struct {
	UserID string `json:"userId"`
}

func (h *Handler) addMember(w http.ResponseWriter, r *http.Request) {
	actor, ok := h.actor(r)
	if !ok {
		httpresponse.Fail(w, r, http.StatusUnauthorized, httpresponse.ErrAuthInvalidToken, "authentication required")
		return
	}
	groupID, err := uuid.Parse(chi.URLParam(r, "groupID"))
	if err != nil {
		httpresponse.Fail(w, r, http.StatusNotFound, httpresponse.ErrResourceNotFound, "group not found")
		return
	}

	var req addMemberRequest
	if err := httpjson.Decode(w, r, &req); err != nil {
		httpresponse.Fail(w, r, http.StatusBadRequest, httpresponse.ErrValidationFailed, "malformed JSON body")
		return
	}
	targetID, err := uuid.Parse(req.UserID)
	if err != nil {
		httpresponse.Fail(w, r, http.StatusBadRequest, httpresponse.ErrValidationFailed, "userId must be a valid UUID")
		return
	}

	targetLevel, exists := h.lookup(r.Context(), targetID)
	if !exists {
		// Do not reveal whether the user exists; treat as a group-scoped
		// not-found.
		httpresponse.Fail(w, r, http.StatusNotFound, httpresponse.ErrResourceNotFound, "group or user not found")
		return
	}

	err = h.svc.AddMember(r.Context(), groupID, targetID, targetLevel, actor)
	switch {
	case errors.Is(err, ErrNotFound):
		httpresponse.Fail(w, r, http.StatusNotFound, httpresponse.ErrResourceNotFound, "group or user not found")
	case errors.Is(err, ErrAlreadyMember):
		httpresponse.Fail(w, r, http.StatusConflict, httpresponse.ErrValidationFailed, "user is already a member")
	case err != nil:
		httpresponse.Fail(w, r, http.StatusInternalServerError, httpresponse.ErrInternal, "failed to add member")
	default:
		httpresponse.OK(w, r, http.StatusCreated, map[string]any{"added": true})
	}
}
