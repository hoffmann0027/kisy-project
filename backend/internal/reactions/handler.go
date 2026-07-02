package reactions

import (
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"kisy-backend/pkg/httpjson"
	"kisy-backend/pkg/httpresponse"
)

// Handler exposes reaction endpoints under a message. RequireAuth is
// applied by the router.
type Handler struct {
	svc   *Service
	actor func(*http.Request) (Actor, bool)
}

func NewHandler(svc *Service, actor func(*http.Request) (Actor, bool)) *Handler {
	return &Handler{svc: svc, actor: actor}
}

func (h *Handler) Routes(r chi.Router) {
	r.Post("/messages/{messageID}/reactions", h.add)
	r.Delete("/messages/{messageID}/reactions", h.remove)
}

type reactionRequest struct {
	Emoji string `json:"emoji"`
}

func (h *Handler) add(w http.ResponseWriter, r *http.Request) {
	h.mutate(w, r, true)
}

func (h *Handler) remove(w http.ResponseWriter, r *http.Request) {
	h.mutate(w, r, false)
}

func (h *Handler) mutate(w http.ResponseWriter, r *http.Request, add bool) {
	actor, ok := h.actor(r)
	if !ok {
		httpresponse.Fail(w, r, http.StatusUnauthorized, httpresponse.ErrAuthInvalidToken, "authentication required")
		return
	}
	messageID, err := uuid.Parse(chi.URLParam(r, "messageID"))
	if err != nil {
		httpresponse.Fail(w, r, http.StatusNotFound, httpresponse.ErrResourceNotFound, "message not found")
		return
	}

	var req reactionRequest
	if err := httpjson.Decode(w, r, &req); err != nil {
		httpresponse.Fail(w, r, http.StatusBadRequest, httpresponse.ErrValidationFailed, "malformed JSON body")
		return
	}

	if add {
		err = h.svc.Add(r.Context(), messageID, req.Emoji, actor)
	} else {
		err = h.svc.Remove(r.Context(), messageID, req.Emoji, actor)
	}

	switch {
	case errors.Is(err, ErrInvalidEmoji):
		httpresponse.Fail(w, r, http.StatusBadRequest, httpresponse.ErrValidationFailed, "emoji is required and must be at most 32 bytes")
	case errors.Is(err, ErrNotFound):
		httpresponse.Fail(w, r, http.StatusNotFound, httpresponse.ErrResourceNotFound, "message not found")
	case err != nil:
		httpresponse.Fail(w, r, http.StatusInternalServerError, httpresponse.ErrInternal, "internal error")
	default:
		httpresponse.OK(w, r, http.StatusOK, map[string]any{"ok": true})
	}
}
