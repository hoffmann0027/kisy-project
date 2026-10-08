package feedback

import (
	"errors"
	"fmt"
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
	r.Post("/{id}/reply", h.reply)
	r.Delete("/{id}", h.delete)
}

// fail maps the service's refusals onto the API contract.
func fail(w http.ResponseWriter, r *http.Request, err error, what string) {
	var limit *DailyLimitError
	switch {
	case errors.As(err, &limit):
		w.Header().Set("Retry-After", strconv.Itoa(int(limit.RetryAfter.Seconds())+1))
		// Written for the screen (QUOTA_EXCEEDED is a user-facing code).
		hours := int(limit.RetryAfter.Hours())
		msg := "Отзыв можно оставлять раз в сутки. Следующий — меньше чем через час"
		if hours >= 1 {
			msg = fmt.Sprintf("Отзыв можно оставлять раз в сутки. Следующий — через %d ч", hours+1)
		}
		httpresponse.Fail(w, r, http.StatusTooManyRequests, httpresponse.ErrQuotaExceeded, msg)
	case errors.Is(err, ErrEmpty):
		httpresponse.Fail(w, r, http.StatusBadRequest, httpresponse.ErrValidationFailed, "text is required")
	case errors.Is(err, ErrTooLong):
		httpresponse.Fail(w, r, http.StatusBadRequest, httpresponse.ErrValidationFailed, "text is too long (max 2000 characters)")
	case errors.Is(err, ErrForbidden):
		httpresponse.Fail(w, r, http.StatusForbidden, httpresponse.ErrAccessDenied, "not permitted")
	case errors.Is(err, ErrNotFound):
		httpresponse.Fail(w, r, http.StatusNotFound, httpresponse.ErrResourceNotFound, "feedback not found")
	case errors.Is(err, ErrAlreadyReplied):
		httpresponse.Fail(w, r, http.StatusConflict, httpresponse.ErrValidationFailed, "this feedback has already been answered")
	default:
		httpresponse.Fail(w, r, http.StatusInternalServerError, httpresponse.ErrInternal, "failed to "+what)
	}
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	actor, ok := h.actor(r)
	if !ok {
		httpresponse.Fail(w, r, http.StatusUnauthorized, httpresponse.ErrAuthInvalidToken, "authentication required")
		return
	}
	limit := 0
	if raw := r.URL.Query().Get("limit"); raw != "" {
		limit, _ = strconv.Atoi(raw)
	}
	page, err := h.svc.List(r.Context(), actor, r.URL.Query().Get("scope"), r.URL.Query().Get("cursor"), limit)
	if err != nil {
		fail(w, r, err, "list feedback")
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
	if err != nil {
		fail(w, r, err, "submit feedback")
		return
	}
	httpresponse.OK(w, r, http.StatusCreated, map[string]any{"feedback": dto})
}

func (h *Handler) reply(w http.ResponseWriter, r *http.Request) {
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
	var req createRequest
	if err := httpjson.Decode(w, r, &req); err != nil {
		httpresponse.Fail(w, r, http.StatusBadRequest, httpresponse.ErrValidationFailed, "malformed JSON body")
		return
	}
	if err := h.svc.Reply(r.Context(), actor, id, req.Body); err != nil {
		fail(w, r, err, "reply")
		return
	}
	httpresponse.OK(w, r, http.StatusOK, map[string]any{"replied": true})
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
	if err := h.svc.Delete(r.Context(), id, actor); err != nil {
		fail(w, r, err, "delete feedback")
		return
	}
	httpresponse.OK(w, r, http.StatusOK, map[string]any{"deleted": true})
}
