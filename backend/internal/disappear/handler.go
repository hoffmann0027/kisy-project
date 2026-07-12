package disappear

import (
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"kisy-backend/pkg/httpjson"
	"kisy-backend/pkg/httpresponse"
)

// Handler exposes the per-chat disappearing-timer endpoints. RequireAuth is
// applied by the router.
type Handler struct {
	svc   *Service
	actor func(*http.Request) (Actor, bool)
}

func NewHandler(svc *Service, actor func(*http.Request) (Actor, bool)) *Handler {
	return &Handler{svc: svc, actor: actor}
}

// ChatRoutes registers under /chats.
func (h *Handler) ChatRoutes(r chi.Router) {
	r.Get("/{chatType}/{chatID}/disappearing", h.get)
	r.Put("/{chatType}/{chatID}/disappearing", h.set)
}

func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	actor, ok := h.actor(r)
	if !ok {
		unauth(w, r)
		return
	}
	chatID, err := uuid.Parse(chi.URLParam(r, "chatID"))
	if err != nil {
		badRequest(w, r, "invalid chat id")
		return
	}
	setting, err := h.svc.Get(r.Context(), chi.URLParam(r, "chatType"), chatID, actor)
	if err != nil {
		fail(w, r, err)
		return
	}
	httpresponse.OK(w, r, http.StatusOK, map[string]any{"disappearing": setting})
}

type setRequest struct {
	// TTLSeconds for new messages; null/0 disables the timer.
	TTLSeconds *int64 `json:"ttlSeconds"`
}

func (h *Handler) set(w http.ResponseWriter, r *http.Request) {
	actor, ok := h.actor(r)
	if !ok {
		unauth(w, r)
		return
	}
	chatID, err := uuid.Parse(chi.URLParam(r, "chatID"))
	if err != nil {
		badRequest(w, r, "invalid chat id")
		return
	}
	var req setRequest
	if err := httpjson.Decode(w, r, &req); err != nil {
		badRequest(w, r, "malformed JSON body")
		return
	}
	setting, err := h.svc.Set(r.Context(), chi.URLParam(r, "chatType"), chatID, req.TTLSeconds, actor)
	if err != nil {
		fail(w, r, err)
		return
	}
	httpresponse.OK(w, r, http.StatusOK, map[string]any{"disappearing": setting})
}

// --- helpers ---

func unauth(w http.ResponseWriter, r *http.Request) {
	httpresponse.Fail(w, r, http.StatusUnauthorized, httpresponse.ErrAuthInvalidToken, "authentication required")
}

func badRequest(w http.ResponseWriter, r *http.Request, msg string) {
	httpresponse.Fail(w, r, http.StatusBadRequest, httpresponse.ErrValidationFailed, msg)
}

func fail(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, ErrValidation):
		badRequest(w, r, "invalid request")
	case errors.Is(err, ErrNotFound):
		httpresponse.Fail(w, r, http.StatusNotFound, httpresponse.ErrResourceNotFound, "not found")
	default:
		httpresponse.Fail(w, r, http.StatusInternalServerError, httpresponse.ErrInternal, "internal error")
	}
}
