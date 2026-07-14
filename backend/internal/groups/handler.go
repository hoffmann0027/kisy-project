package groups

import (
	"context"
	"errors"
	"io"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"kisy-backend/pkg/httpjson"
	"kisy-backend/pkg/httpresponse"
)

// maxAvatarUpload bounds the group-avatar request body (validation happens in
// the avatars service; this is a coarse transport guard).
const maxAvatarUpload = 1 << 20 // 1 MiB

// UserLookup resolves a target user's clearance level, injected by the
// main router to avoid a groups→users import cycle. It returns false when
// the user does not exist.
type UserLookup func(ctx context.Context, id uuid.UUID) (level int, ok bool)

// AvatarStore stores a validated avatar image and returns its versioned URL.
// Satisfied by *avatars.Service; a local interface avoids a groups→avatars
// dependency in the domain.
type AvatarStore interface {
	Store(ctx context.Context, ownerType string, ownerID uuid.UUID, raw []byte) (string, error)
}

// Handler exposes /groups endpoints. RequireAuth is applied by the router.
type Handler struct {
	svc     *Service
	avatars AvatarStore
	actor   func(*http.Request) (ActorMeta, bool)
	lookup  UserLookup
}

func NewHandler(svc *Service, avatars AvatarStore, actor func(*http.Request) (ActorMeta, bool), lookup UserLookup) *Handler {
	return &Handler{svc: svc, avatars: avatars, actor: actor, lookup: lookup}
}

func (h *Handler) Routes(r chi.Router) {
	r.Get("/", h.list)
	r.Post("/", h.create)            // any user may create (clearance validated in service)
	r.Get("/directory", h.directory) // groups the actor may join
	r.Get("/{groupID}", h.get)
	r.Patch("/{groupID}", h.update)            // CEO-only: change the group's level
	r.Patch("/{groupID}/settings", h.settings) // owner/CEO: join/post policy
	r.Delete("/{groupID}", h.delete)
	r.Get("/{groupID}/me", h.viewer) // caller's membership/role/post-right
	r.Get("/{groupID}/members", h.listMembers)
	r.Post("/{groupID}/members", h.addMember)
	r.Post("/{groupID}/members/{userID}/role", h.setMemberRole) // owner/CEO
	r.Post("/{groupID}/avatar", h.uploadAvatar)
	r.Post("/{groupID}/join", h.join)
	r.Get("/{groupID}/requests", h.listRequests)
	r.Post("/{groupID}/requests/{userID}/approve", h.approveRequest)
	r.Post("/{groupID}/requests/{userID}/reject", h.rejectRequest)
}

// uploadAvatar accepts a raw image body and sets the group avatar. Only the
// founder or the CEO may do this (enforced in the service).
func (h *Handler) uploadAvatar(w http.ResponseWriter, r *http.Request) {
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

	raw, err := io.ReadAll(http.MaxBytesReader(w, r.Body, maxAvatarUpload))
	if err != nil {
		httpresponse.Fail(w, r, http.StatusRequestEntityTooLarge, httpresponse.ErrValidationFailed, "image too large")
		return
	}

	url, err := h.avatars.Store(r.Context(), "group", groupID, raw)
	if err != nil {
		httpresponse.Fail(w, r, http.StatusBadRequest, httpresponse.ErrValidationFailed, "invalid image: "+err.Error())
		return
	}

	g, err := h.svc.SetAvatar(r.Context(), groupID, url, actor)
	switch {
	case errors.Is(err, ErrNotFound):
		httpresponse.Fail(w, r, http.StatusNotFound, httpresponse.ErrResourceNotFound, "group not found")
	case errors.Is(err, ErrForbidden):
		httpresponse.Fail(w, r, http.StatusForbidden, httpresponse.ErrAccessDenied, "only the CEO or the group founder may change the avatar")
	case err != nil:
		httpresponse.Fail(w, r, http.StatusInternalServerError, httpresponse.ErrInternal, "internal error")
	default:
		httpresponse.OK(w, r, http.StatusOK, map[string]any{"group": g.ToDTO()})
	}
}

type updateRequest struct {
	MinRoleLevel int `json:"minRoleLevel"`
}

// update changes a group's minimum clearance level after creation. Only the
// CEO may do this (enforced in the service).
func (h *Handler) update(w http.ResponseWriter, r *http.Request) {
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

	var req updateRequest
	if err := httpjson.Decode(w, r, &req); err != nil {
		httpresponse.Fail(w, r, http.StatusBadRequest, httpresponse.ErrValidationFailed, "malformed JSON body")
		return
	}
	if req.MinRoleLevel < 1 || req.MinRoleLevel > 10 {
		httpresponse.Fail(w, r, http.StatusBadRequest, httpresponse.ErrValidationFailed, "minRoleLevel must be between 1 and 10")
		return
	}

	g, err := h.svc.SetMinRoleLevel(r.Context(), id, req.MinRoleLevel, actor)
	switch {
	case errors.Is(err, ErrNotFound):
		httpresponse.Fail(w, r, http.StatusNotFound, httpresponse.ErrResourceNotFound, "group not found")
	case errors.Is(err, ErrForbidden):
		httpresponse.Fail(w, r, http.StatusForbidden, httpresponse.ErrAccessDenied, "только CEO может менять уровень группы")
	case err != nil:
		httpresponse.Fail(w, r, http.StatusInternalServerError, httpresponse.ErrInternal, "failed to update group")
	default:
		httpresponse.OK(w, r, http.StatusOK, map[string]any{"group": g.ToDTO()})
	}
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

// --- Stage N: access settings, directory, join, requests, roles ---

// reqActorGroup resolves the actor and the {groupID} path param, writing the
// error response itself and returning ok=false on failure.
func (h *Handler) reqActorGroup(w http.ResponseWriter, r *http.Request) (ActorMeta, uuid.UUID, bool) {
	actor, ok := h.actor(r)
	if !ok {
		httpresponse.Fail(w, r, http.StatusUnauthorized, httpresponse.ErrAuthInvalidToken, "authentication required")
		return ActorMeta{}, uuid.Nil, false
	}
	groupID, err := uuid.Parse(chi.URLParam(r, "groupID"))
	if err != nil {
		httpresponse.Fail(w, r, http.StatusNotFound, httpresponse.ErrResourceNotFound, "group not found")
		return ActorMeta{}, uuid.Nil, false
	}
	return actor, groupID, true
}

type settingsRequest struct {
	JoinPolicy string `json:"joinPolicy"`
	PostPolicy string `json:"postPolicy"`
}

func (h *Handler) settings(w http.ResponseWriter, r *http.Request) {
	actor, groupID, ok := h.reqActorGroup(w, r)
	if !ok {
		return
	}
	var req settingsRequest
	if err := httpjson.Decode(w, r, &req); err != nil {
		httpresponse.Fail(w, r, http.StatusBadRequest, httpresponse.ErrValidationFailed, "malformed JSON body")
		return
	}
	g, err := h.svc.SetPolicies(r.Context(), groupID, req.JoinPolicy, req.PostPolicy, actor)
	switch {
	case errors.Is(err, ErrBadPolicy):
		httpresponse.Fail(w, r, http.StatusBadRequest, httpresponse.ErrValidationFailed, "joinPolicy must be open|request and postPolicy all|editors")
	case errors.Is(err, ErrNotFound):
		httpresponse.Fail(w, r, http.StatusNotFound, httpresponse.ErrResourceNotFound, "group not found")
	case errors.Is(err, ErrForbidden):
		httpresponse.Fail(w, r, http.StatusForbidden, httpresponse.ErrAccessDenied, "only the owner or CEO may change access settings")
	case err != nil:
		httpresponse.Fail(w, r, http.StatusInternalServerError, httpresponse.ErrInternal, "failed to update settings")
	default:
		httpresponse.OK(w, r, http.StatusOK, map[string]any{"group": g.ToDTO()})
	}
}

func (h *Handler) directory(w http.ResponseWriter, r *http.Request) {
	actor, ok := h.actor(r)
	if !ok {
		httpresponse.Fail(w, r, http.StatusUnauthorized, httpresponse.ErrAuthInvalidToken, "authentication required")
		return
	}
	entries, err := h.svc.Directory(r.Context(), actor)
	if err != nil {
		httpresponse.Fail(w, r, http.StatusInternalServerError, httpresponse.ErrInternal, "failed to list directory")
		return
	}
	out := make([]DirectoryDTO, 0, len(entries))
	for i := range entries {
		out = append(out, DirectoryDTO{DTO: entries[i].Group.ToDTO(), RequestStatus: entries[i].RequestStatus})
	}
	httpresponse.OK(w, r, http.StatusOK, map[string]any{"groups": out})
}

func (h *Handler) join(w http.ResponseWriter, r *http.Request) {
	actor, groupID, ok := h.reqActorGroup(w, r)
	if !ok {
		return
	}
	res, err := h.svc.Join(r.Context(), groupID, actor)
	switch {
	case errors.Is(err, ErrNotFound):
		httpresponse.Fail(w, r, http.StatusNotFound, httpresponse.ErrResourceNotFound, "group not found")
	case err != nil:
		httpresponse.Fail(w, r, http.StatusInternalServerError, httpresponse.ErrInternal, "failed to join")
	default:
		httpresponse.OK(w, r, http.StatusOK, map[string]any{"joined": res.Joined, "status": res.Status})
	}
}

func (h *Handler) listRequests(w http.ResponseWriter, r *http.Request) {
	actor, groupID, ok := h.reqActorGroup(w, r)
	if !ok {
		return
	}
	reqs, err := h.svc.ListRequests(r.Context(), groupID, actor)
	switch {
	case errors.Is(err, ErrNotFound):
		httpresponse.Fail(w, r, http.StatusNotFound, httpresponse.ErrResourceNotFound, "group not found")
	case errors.Is(err, ErrForbidden):
		httpresponse.Fail(w, r, http.StatusForbidden, httpresponse.ErrAccessDenied, "only editors, the owner or CEO may view requests")
	case err != nil:
		httpresponse.Fail(w, r, http.StatusInternalServerError, httpresponse.ErrInternal, "internal error")
	default:
		httpresponse.OK(w, r, http.StatusOK, map[string]any{"requests": reqs})
	}
}

// reqTargetUser resolves the {userID} path param and its clearance level.
func (h *Handler) reqTargetUser(w http.ResponseWriter, r *http.Request) (uuid.UUID, int, bool) {
	targetID, err := uuid.Parse(chi.URLParam(r, "userID"))
	if err != nil {
		httpresponse.Fail(w, r, http.StatusNotFound, httpresponse.ErrResourceNotFound, "group or user not found")
		return uuid.Nil, 0, false
	}
	level, exists := h.lookup(r.Context(), targetID)
	if !exists {
		httpresponse.Fail(w, r, http.StatusNotFound, httpresponse.ErrResourceNotFound, "group or user not found")
		return uuid.Nil, 0, false
	}
	return targetID, level, true
}

func (h *Handler) approveRequest(w http.ResponseWriter, r *http.Request) {
	actor, groupID, ok := h.reqActorGroup(w, r)
	if !ok {
		return
	}
	targetID, targetLevel, ok := h.reqTargetUser(w, r)
	if !ok {
		return
	}
	err := h.svc.ApproveRequest(r.Context(), groupID, targetID, targetLevel, actor)
	switch {
	case errors.Is(err, ErrNotFound), errors.Is(err, ErrRequestNotFound):
		httpresponse.Fail(w, r, http.StatusNotFound, httpresponse.ErrResourceNotFound, "request not found")
	case errors.Is(err, ErrForbidden):
		httpresponse.Fail(w, r, http.StatusForbidden, httpresponse.ErrAccessDenied, "not permitted")
	case err != nil:
		httpresponse.Fail(w, r, http.StatusInternalServerError, httpresponse.ErrInternal, "failed to approve")
	default:
		httpresponse.OK(w, r, http.StatusOK, map[string]any{"approved": true})
	}
}

func (h *Handler) rejectRequest(w http.ResponseWriter, r *http.Request) {
	actor, groupID, ok := h.reqActorGroup(w, r)
	if !ok {
		return
	}
	targetID, _, ok := h.reqTargetUser(w, r)
	if !ok {
		return
	}
	err := h.svc.RejectRequest(r.Context(), groupID, targetID, actor)
	switch {
	case errors.Is(err, ErrNotFound), errors.Is(err, ErrRequestNotFound):
		httpresponse.Fail(w, r, http.StatusNotFound, httpresponse.ErrResourceNotFound, "request not found")
	case errors.Is(err, ErrForbidden):
		httpresponse.Fail(w, r, http.StatusForbidden, httpresponse.ErrAccessDenied, "not permitted")
	case err != nil:
		httpresponse.Fail(w, r, http.StatusInternalServerError, httpresponse.ErrInternal, "failed to reject")
	default:
		httpresponse.OK(w, r, http.StatusOK, map[string]any{"rejected": true})
	}
}

func (h *Handler) viewer(w http.ResponseWriter, r *http.Request) {
	actor, groupID, ok := h.reqActorGroup(w, r)
	if !ok {
		return
	}
	vs, err := h.svc.Viewer(r.Context(), groupID, actor)
	switch {
	case errors.Is(err, ErrNotFound):
		httpresponse.Fail(w, r, http.StatusNotFound, httpresponse.ErrResourceNotFound, "group not found")
	case err != nil:
		httpresponse.Fail(w, r, http.StatusInternalServerError, httpresponse.ErrInternal, "internal error")
	default:
		httpresponse.OK(w, r, http.StatusOK, map[string]any{"viewer": vs})
	}
}

type setRoleRequest struct {
	Role string `json:"role"`
}

func (h *Handler) setMemberRole(w http.ResponseWriter, r *http.Request) {
	actor, groupID, ok := h.reqActorGroup(w, r)
	if !ok {
		return
	}
	targetID, err := uuid.Parse(chi.URLParam(r, "userID"))
	if err != nil {
		httpresponse.Fail(w, r, http.StatusNotFound, httpresponse.ErrResourceNotFound, "group or user not found")
		return
	}
	var req setRoleRequest
	if err := httpjson.Decode(w, r, &req); err != nil {
		httpresponse.Fail(w, r, http.StatusBadRequest, httpresponse.ErrValidationFailed, "malformed JSON body")
		return
	}
	err = h.svc.SetMemberRole(r.Context(), groupID, targetID, req.Role, actor)
	switch {
	case errors.Is(err, ErrBadPolicy):
		httpresponse.Fail(w, r, http.StatusBadRequest, httpresponse.ErrValidationFailed, "role must be member|editor|moderator")
	case errors.Is(err, ErrNotFound), errors.Is(err, ErrNotMember):
		httpresponse.Fail(w, r, http.StatusNotFound, httpresponse.ErrResourceNotFound, "group or member not found")
	case errors.Is(err, ErrForbidden):
		httpresponse.Fail(w, r, http.StatusForbidden, httpresponse.ErrAccessDenied, "only the owner or CEO may change roles")
	case err != nil:
		httpresponse.Fail(w, r, http.StatusInternalServerError, httpresponse.ErrInternal, "failed to change role")
	default:
		httpresponse.OK(w, r, http.StatusOK, map[string]any{"ok": true})
	}
}
