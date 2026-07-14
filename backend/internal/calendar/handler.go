package calendar

import (
	"errors"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"kisy-backend/pkg/httpjson"
	"kisy-backend/pkg/httpresponse"
)

// Handler exposes calendar endpoints. RequireAuth is applied by the router.
type Handler struct {
	svc   *Service
	actor func(*http.Request) (Actor, bool)
}

func NewHandler(svc *Service, actor func(*http.Request) (Actor, bool)) *Handler {
	return &Handler{svc: svc, actor: actor}
}

// GroupRoutes are mounted under /groups.
func (h *Handler) GroupRoutes(r chi.Router) {
	r.Get("/{groupID}/calendar", h.list)
	r.Post("/{groupID}/calendar", h.create)
}

// EventRoutes are mounted under /calendar.
func (h *Handler) EventRoutes(r chi.Router) {
	r.Patch("/{eventID}", h.update)
	r.Delete("/{eventID}", h.delete)
}

func (h *Handler) reqActor(w http.ResponseWriter, r *http.Request) (Actor, bool) {
	actor, ok := h.actor(r)
	if !ok {
		httpresponse.Fail(w, r, http.StatusUnauthorized, httpresponse.ErrAuthInvalidToken, "authentication required")
		return Actor{}, false
	}
	return actor, true
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	actor, ok := h.reqActor(w, r)
	if !ok {
		return
	}
	groupID, err := uuid.Parse(chi.URLParam(r, "groupID"))
	if err != nil {
		httpresponse.Fail(w, r, http.StatusNotFound, httpresponse.ErrResourceNotFound, "group not found")
		return
	}
	from, err1 := time.Parse(time.RFC3339, r.URL.Query().Get("from"))
	to, err2 := time.Parse(time.RFC3339, r.URL.Query().Get("to"))
	if err1 != nil || err2 != nil || !to.After(from) {
		httpresponse.Fail(w, r, http.StatusBadRequest, httpresponse.ErrValidationFailed, "from/to must be RFC3339 with to > from")
		return
	}

	view, err := h.svc.List(r.Context(), groupID, from, to, actor)
	switch {
	case errors.Is(err, ErrNotFound):
		httpresponse.Fail(w, r, http.StatusNotFound, httpresponse.ErrResourceNotFound, "group not found")
	case errors.Is(err, ErrForbidden):
		httpresponse.Fail(w, r, http.StatusForbidden, httpresponse.ErrAccessDenied, "not a member")
	case err != nil:
		httpresponse.Fail(w, r, http.StatusInternalServerError, httpresponse.ErrInternal, "internal error")
	default:
		httpresponse.OK(w, r, http.StatusOK, map[string]any{"events": view.Events, "cards": view.Cards})
	}
}

type writeRequest struct {
	Title    string     `json:"title"`
	StartsAt time.Time  `json:"startsAt"`
	EndsAt   *time.Time `json:"endsAt"`
	Color    string     `json:"color"`
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	actor, ok := h.reqActor(w, r)
	if !ok {
		return
	}
	groupID, err := uuid.Parse(chi.URLParam(r, "groupID"))
	if err != nil {
		httpresponse.Fail(w, r, http.StatusNotFound, httpresponse.ErrResourceNotFound, "group not found")
		return
	}
	var req writeRequest
	if err := httpjson.Decode(w, r, &req); err != nil {
		httpresponse.Fail(w, r, http.StatusBadRequest, httpresponse.ErrValidationFailed, "malformed JSON body")
		return
	}
	e, err := h.svc.Create(r.Context(), groupID, Input(req), actor)
	h.writeResult(w, r, e, err, http.StatusCreated)
}

func (h *Handler) update(w http.ResponseWriter, r *http.Request) {
	actor, ok := h.reqActor(w, r)
	if !ok {
		return
	}
	eventID, err := uuid.Parse(chi.URLParam(r, "eventID"))
	if err != nil {
		httpresponse.Fail(w, r, http.StatusNotFound, httpresponse.ErrResourceNotFound, "event not found")
		return
	}
	var req writeRequest
	if err := httpjson.Decode(w, r, &req); err != nil {
		httpresponse.Fail(w, r, http.StatusBadRequest, httpresponse.ErrValidationFailed, "malformed JSON body")
		return
	}
	e, err := h.svc.Update(r.Context(), eventID, Input(req), actor)
	h.writeResult(w, r, e, err, http.StatusOK)
}

func (h *Handler) delete(w http.ResponseWriter, r *http.Request) {
	actor, ok := h.reqActor(w, r)
	if !ok {
		return
	}
	eventID, err := uuid.Parse(chi.URLParam(r, "eventID"))
	if err != nil {
		httpresponse.Fail(w, r, http.StatusNotFound, httpresponse.ErrResourceNotFound, "event not found")
		return
	}
	err = h.svc.Delete(r.Context(), eventID, actor)
	switch {
	case errors.Is(err, ErrNotFound):
		httpresponse.Fail(w, r, http.StatusNotFound, httpresponse.ErrResourceNotFound, "event not found")
	case errors.Is(err, ErrForbidden):
		httpresponse.Fail(w, r, http.StatusForbidden, httpresponse.ErrAccessDenied, "not permitted")
	case err != nil:
		httpresponse.Fail(w, r, http.StatusInternalServerError, httpresponse.ErrInternal, "failed to delete")
	default:
		httpresponse.OK(w, r, http.StatusOK, map[string]any{"deleted": true})
	}
}

// writeResult maps create/update errors and emits the event DTO on success.
func (h *Handler) writeResult(w http.ResponseWriter, r *http.Request, e *Event, err error, okStatus int) {
	switch {
	case errors.Is(err, ErrNotFound):
		httpresponse.Fail(w, r, http.StatusNotFound, httpresponse.ErrResourceNotFound, "not found")
	case errors.Is(err, ErrForbidden):
		httpresponse.Fail(w, r, http.StatusForbidden, httpresponse.ErrAccessDenied, "not permitted")
	case errors.Is(err, ErrBadColor):
		httpresponse.Fail(w, r, http.StatusBadRequest, httpresponse.ErrValidationFailed, "color must be one of the palette")
	case errors.Is(err, ErrBadTimeRange):
		httpresponse.Fail(w, r, http.StatusBadRequest, httpresponse.ErrValidationFailed, "endsAt must be at or after startsAt")
	case errors.Is(err, ErrValidation):
		httpresponse.Fail(w, r, http.StatusBadRequest, httpresponse.ErrValidationFailed, "title is required (1-200 chars)")
	case err != nil:
		httpresponse.Fail(w, r, http.StatusInternalServerError, httpresponse.ErrInternal, "internal error")
	default:
		httpresponse.OK(w, r, okStatus, map[string]any{"event": e.ToDTO()})
	}
}
