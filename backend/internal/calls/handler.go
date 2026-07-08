package calls

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"kisy-backend/pkg/httpresponse"
)

// Handler exposes the calls REST surface. RequireAuth is applied by the router;
// the live signaling itself runs over the WebSocket gateway, not here.
type Handler struct {
	svc   *Service
	actor func(*http.Request) (Actor, bool)
}

func NewHandler(svc *Service, actor func(*http.Request) (Actor, bool)) *Handler {
	return &Handler{svc: svc, actor: actor}
}

func (h *Handler) Routes(r chi.Router) {
	r.Get("/ice-config", h.iceConfig)
	r.Get("/history", h.history)
}

func (h *Handler) iceConfig(w http.ResponseWriter, r *http.Request) {
	actor, ok := h.actor(r)
	if !ok {
		httpresponse.Fail(w, r, http.StatusUnauthorized, httpresponse.ErrAuthInvalidToken, "authentication required")
		return
	}
	httpresponse.OK(w, r, http.StatusOK, h.svc.ICEConfig(actor))
}

func (h *Handler) history(w http.ResponseWriter, r *http.Request) {
	actor, ok := h.actor(r)
	if !ok {
		httpresponse.Fail(w, r, http.StatusUnauthorized, httpresponse.ErrAuthInvalidToken, "authentication required")
		return
	}
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	items, err := h.svc.History(r.Context(), actor, limit, offset)
	if err != nil {
		httpresponse.Fail(w, r, http.StatusInternalServerError, httpresponse.ErrInternal, "internal error")
		return
	}
	httpresponse.OK(w, r, http.StatusOK, map[string]any{"calls": items})
}
