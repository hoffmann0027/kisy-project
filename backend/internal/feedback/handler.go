package feedback

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"kisy-backend/pkg/httpjson"
	"kisy-backend/pkg/httpresponse"
)

// Handler exposes /feedback. RequireAuth is applied by the router.
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
	r.Delete("/{id}", h.delete)
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	if _, ok := h.actor(r); !ok {
		httpresponse.Fail(w, r, http.StatusUnauthorized, httpresponse.ErrAuthInvalidToken, "authentication required")
		return
	}
	limit := 0
	if raw := r.URL.Query().Get("limit"); raw != "" {
		limit, _ = strconv.Atoi(raw)
	}
	page, err := h.svc.List(r.Context(), r.URL.Query().Get("cursor"), limit)
	if err != nil {
		httpresponse.Fail(w, r, http.StatusInternalServerError, httpresponse.ErrInternal, "failed to list feedback")
		return
	}
	httpresponse.OK(w, r, http.StatusOK, page)
}

type createRequest struct {
	Body string `json:"body"`
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
	dto, err := h.svc.Create(r.Context(), actor.UserID, req.Body)
	switch {
	case errors.Is(err, ErrEmpty):
		httpresponse.Fail(w, r, http.StatusBadRequest, httpresponse.ErrValidationFailed, "feedback text is required")
	case errors.Is(err, ErrTooLong):
		httpresponse.Fail(w, r, http.StatusBadRequest, httpresponse.ErrValidationFailed, "feedback is too long (max 2000 characters)")
	case err != nil:
		httpresponse.Fail(w, r, http.StatusInternalServerError, httpresponse.ErrInternal, "failed to submit feedback")
	default:
		httpresponse.OK(w, r, http.StatusCreated, map[string]any{"feedback": dto})
	}
}

func (h *Handler) delete(w http.ResponseWriter, r *http.Request) {
	actor, ok := h.actor(r)
	if !ok {
		httpresponse.Fail(w, r, http.StatusUnauthorized, httpresponse.ErrAuthInvalidToken, "authentication required")
		return
	}
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httpresponse.Fail(w, r, http.StatusNotFound, httpresponse.ErrResourceNotFound, "feedback not found")
		return
	}
	err = h.svc.Delete(r.Context(), id, actor)
	switch {
	case errors.Is(err, ErrForbidden):
		httpresponse.Fail(w, r, http.StatusForbidden, httpresponse.ErrAccessDenied, "only the CEO may delete feedback")
	case errors.Is(err, ErrNotFound):
		httpresponse.Fail(w, r, http.StatusNotFound, httpresponse.ErrResourceNotFound, "feedback not found")
	case err != nil:
		httpresponse.Fail(w, r, http.StatusInternalServerError, httpresponse.ErrInternal, "failed to delete feedback")
	default:
		httpresponse.OK(w, r, http.StatusOK, map[string]any{"deleted": true})
	}
}
