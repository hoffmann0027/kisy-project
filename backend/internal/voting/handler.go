package voting

import (
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"kisy-backend/pkg/httpjson"
	"kisy-backend/pkg/httpresponse"
)

// Handler exposes the voting endpoints. RequireAuth is applied by the router;
// CEO-only operations are enforced in the service.
type Handler struct {
	svc   *Service
	actor func(*http.Request) (Actor, bool)
}

func NewHandler(svc *Service, actor func(*http.Request) (Actor, bool)) *Handler {
	return &Handler{svc: svc, actor: actor}
}

func (h *Handler) Routes(r chi.Router) {
	r.Get("/", h.list)
	r.Post("/", h.create)
	r.Post("/{id}/close", h.close)
	r.Delete("/{id}", h.delete)
	r.Post("/options/{optionID}/vote", h.vote)
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	actor, ok := h.actor(r)
	if !ok {
		unauth(w, r)
		return
	}
	polls, err := h.svc.List(r.Context(), actor)
	if err != nil {
		fail(w, r, err)
		return
	}
	httpresponse.OK(w, r, http.StatusOK, map[string]any{"polls": polls})
}

type createRequest struct {
	Question string   `json:"question"`
	Options  []string `json:"options"`
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	actor, ok := h.actor(r)
	if !ok {
		unauth(w, r)
		return
	}
	var req createRequest
	if err := httpjson.Decode(w, r, &req); err != nil {
		httpresponse.Fail(w, r, http.StatusBadRequest, httpresponse.ErrValidationFailed, "malformed JSON body")
		return
	}
	id, err := h.svc.Create(r.Context(), req.Question, req.Options, actor)
	if err != nil {
		fail(w, r, err)
		return
	}
	httpresponse.OK(w, r, http.StatusCreated, map[string]any{"id": id})
}

func (h *Handler) vote(w http.ResponseWriter, r *http.Request) {
	actor, ok := h.actor(r)
	if !ok {
		unauth(w, r)
		return
	}
	optionID, err := uuid.Parse(chi.URLParam(r, "optionID"))
	if err != nil {
		httpresponse.Fail(w, r, http.StatusNotFound, httpresponse.ErrResourceNotFound, "option not found")
		return
	}
	if err := h.svc.Vote(r.Context(), optionID, actor); err != nil {
		fail(w, r, err)
		return
	}
	httpresponse.OK(w, r, http.StatusOK, map[string]any{"voted": true})
}

func (h *Handler) close(w http.ResponseWriter, r *http.Request) {
	actor, ok := h.actor(r)
	if !ok {
		unauth(w, r)
		return
	}
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httpresponse.Fail(w, r, http.StatusNotFound, httpresponse.ErrResourceNotFound, "poll not found")
		return
	}
	if err := h.svc.Close(r.Context(), id, actor); err != nil {
		fail(w, r, err)
		return
	}
	httpresponse.OK(w, r, http.StatusOK, map[string]any{"closed": true})
}

func (h *Handler) delete(w http.ResponseWriter, r *http.Request) {
	actor, ok := h.actor(r)
	if !ok {
		unauth(w, r)
		return
	}
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httpresponse.Fail(w, r, http.StatusNotFound, httpresponse.ErrResourceNotFound, "poll not found")
		return
	}
	if err := h.svc.Delete(r.Context(), id, actor); err != nil {
		fail(w, r, err)
		return
	}
	httpresponse.OK(w, r, http.StatusOK, map[string]any{"deleted": true})
}

func unauth(w http.ResponseWriter, r *http.Request) {
	httpresponse.Fail(w, r, http.StatusUnauthorized, httpresponse.ErrAuthInvalidToken, "authentication required")
}

func fail(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, ErrForbidden):
		httpresponse.Fail(w, r, http.StatusForbidden, httpresponse.ErrAccessDenied, "not permitted")
	case errors.Is(err, ErrNotFound):
		httpresponse.Fail(w, r, http.StatusNotFound, httpresponse.ErrResourceNotFound, "not found")
	case errors.Is(err, ErrClosed):
		httpresponse.Fail(w, r, http.StatusConflict, httpresponse.ErrValidationFailed, "poll is closed")
	case errors.Is(err, ErrValidation):
		httpresponse.Fail(w, r, http.StatusBadRequest, httpresponse.ErrValidationFailed, "invalid input")
	default:
		httpresponse.Fail(w, r, http.StatusInternalServerError, httpresponse.ErrInternal, "internal error")
	}
}
