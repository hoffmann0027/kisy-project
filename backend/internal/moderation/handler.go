package moderation

import (
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"kisy-backend/pkg/httpjson"
	"kisy-backend/pkg/httpresponse"
)

type Handler struct {
	svc   *Service
	actor func(*http.Request) (ActorMeta, bool)
}

func NewHandler(svc *Service, actor func(*http.Request) (ActorMeta, bool)) *Handler {
	return &Handler{svc: svc, actor: actor}
}

// AdminRoutes are mounted under /admin, behind its CEO gate (admin.Mount).
func (h *Handler) AdminRoutes(r chi.Router) {
	r.Get("/communities", h.listGroups)
	r.Get("/communities/deleted", h.listDeleted)
	r.Get("/communities/{groupID}/sanctions", h.history)
	r.Post("/communities/{groupID}/sanctions", h.issue)
	r.Post("/communities/{groupID}/restore", h.restore)
	r.Post("/sanctions/{sanctionID}/revoke", h.revoke)
}

// GroupRoutes hang off /groups/{groupID}: the banner a group's own editors see.
func (h *Handler) GroupRoutes(r chi.Router) {
	r.Get("/{groupID}/sanctions", h.active)
}

func (h *Handler) listGroups(w http.ResponseWriter, r *http.Request) {
	if _, ok := h.actor(r); !ok {
		unauthorized(w, r)
		return
	}
	list, err := h.svc.ListGroups(r.Context(), r.URL.Query().Get("q"))
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httpresponse.OK(w, r, http.StatusOK, map[string]any{"groups": list})
}

func (h *Handler) listDeleted(w http.ResponseWriter, r *http.Request) {
	if _, ok := h.actor(r); !ok {
		unauthorized(w, r)
		return
	}
	list, err := h.svc.ListDeleted(r.Context())
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httpresponse.OK(w, r, http.StatusOK, map[string]any{"groups": list})
}

func (h *Handler) history(w http.ResponseWriter, r *http.Request) {
	if _, ok := h.actor(r); !ok {
		unauthorized(w, r)
		return
	}
	groupID, ok := pathID(w, r, "groupID")
	if !ok {
		return
	}
	list, err := h.svc.History(r.Context(), groupID)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httpresponse.OK(w, r, http.StatusOK, map[string]any{"sanctions": list})
}

type issueRequest struct {
	Kind     string `json:"kind"`
	Reason   string `json:"reason"`
	Duration string `json:"duration"`
}

func (h *Handler) issue(w http.ResponseWriter, r *http.Request) {
	actor, ok := h.actor(r)
	if !ok {
		unauthorized(w, r)
		return
	}
	groupID, ok := pathID(w, r, "groupID")
	if !ok {
		return
	}
	var req issueRequest
	if err := httpjson.Decode(w, r, &req); err != nil {
		httpresponse.Fail(w, r, http.StatusBadRequest, httpresponse.ErrValidationFailed, "malformed JSON body")
		return
	}
	out, err := h.svc.Issue(r.Context(), IssueInput{GroupID: groupID, Kind: req.Kind, Reason: req.Reason, Duration: req.Duration}, actor)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httpresponse.OK(w, r, http.StatusCreated, out)
}

func (h *Handler) restore(w http.ResponseWriter, r *http.Request) {
	actor, ok := h.actor(r)
	if !ok {
		unauthorized(w, r)
		return
	}
	groupID, ok := pathID(w, r, "groupID")
	if !ok {
		return
	}
	if err := h.svc.Restore(r.Context(), groupID, actor); err != nil {
		h.fail(w, r, err)
		return
	}
	httpresponse.OK(w, r, http.StatusOK, map[string]any{"restored": true})
}

type revokeRequest struct {
	Note string `json:"note"`
}

func (h *Handler) revoke(w http.ResponseWriter, r *http.Request) {
	actor, ok := h.actor(r)
	if !ok {
		unauthorized(w, r)
		return
	}
	sanctionID, ok := pathID(w, r, "sanctionID")
	if !ok {
		return
	}
	var req revokeRequest
	if r.ContentLength != 0 {
		if err := httpjson.Decode(w, r, &req); err != nil {
			httpresponse.Fail(w, r, http.StatusBadRequest, httpresponse.ErrValidationFailed, "malformed JSON body")
			return
		}
	}
	s, err := h.svc.Revoke(r.Context(), sanctionID, req.Note, actor)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httpresponse.OK(w, r, http.StatusOK, map[string]any{"sanction": s})
}

func (h *Handler) active(w http.ResponseWriter, r *http.Request) {
	actor, ok := h.actor(r)
	if !ok {
		unauthorized(w, r)
		return
	}
	groupID, ok := pathID(w, r, "groupID")
	if !ok {
		return
	}
	res, err := h.svc.ActiveFor(r.Context(), groupID, actor)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httpresponse.OK(w, r, http.StatusOK, res)
}

func (h *Handler) fail(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, ErrNotFound):
		httpresponse.Fail(w, r, http.StatusNotFound, httpresponse.ErrResourceNotFound, "not found")
	case errors.Is(err, ErrForbidden):
		httpresponse.Fail(w, r, http.StatusForbidden, httpresponse.ErrAccessDenied, "not permitted")
	case errors.Is(err, ErrReasonRequired):
		httpresponse.Fail(w, r, http.StatusBadRequest, httpresponse.ErrValidationFailed, "Укажите причину")
	case errors.Is(err, ErrReasonTooLong):
		httpresponse.Fail(w, r, http.StatusBadRequest, httpresponse.ErrValidationFailed, "Причина — не длиннее 1000 символов")
	case errors.Is(err, ErrInvalidKind), errors.Is(err, ErrInvalidDuration):
		httpresponse.Fail(w, r, http.StatusBadRequest, httpresponse.ErrValidationFailed, err.Error())
	case errors.Is(err, ErrGroupDeleted):
		httpresponse.Fail(w, r, http.StatusConflict, httpresponse.ErrValidationFailed, "Сообщество удалено — сначала восстановите его")
	case errors.Is(err, ErrNotDeleted):
		httpresponse.Fail(w, r, http.StatusConflict, httpresponse.ErrValidationFailed, "Сообщество не удалено")
	case errors.Is(err, ErrNotRevocable):
		httpresponse.Fail(w, r, http.StatusConflict, httpresponse.ErrValidationFailed, "Эту санкцию нельзя снять")
	case errors.Is(err, ErrRestoreExpired):
		httpresponse.Fail(w, r, http.StatusGone, httpresponse.ErrResourceNotFound, "Срок восстановления истёк")
	default:
		httpresponse.Fail(w, r, http.StatusInternalServerError, httpresponse.ErrInternal, "internal error")
	}
}

func unauthorized(w http.ResponseWriter, r *http.Request) {
	httpresponse.Fail(w, r, http.StatusUnauthorized, httpresponse.ErrAuthInvalidToken, "authentication required")
}

func pathID(w http.ResponseWriter, r *http.Request, name string) (uuid.UUID, bool) {
	id, err := uuid.Parse(chi.URLParam(r, name))
	if err != nil {
		httpresponse.Fail(w, r, http.StatusNotFound, httpresponse.ErrResourceNotFound, "not found")
		return uuid.Nil, false
	}
	return id, true
}
