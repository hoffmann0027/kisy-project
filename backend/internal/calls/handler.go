package calls

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

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
	// Both exist for the phone woken by a call push: it has no socket yet.
	r.Get("/pending", h.pending)
	r.Post("/{callID}/reject", h.reject)
}

// pending answers with the call ringing for the caller of this request, so a
// freshly woken app can show it. Empty rather than 404 when there is none:
// "nobody is calling you" is a normal answer, not an error.
func (h *Handler) pending(w http.ResponseWriter, r *http.Request) {
	actor, ok := h.actor(r)
	if !ok {
		httpresponse.Fail(w, r, http.StatusUnauthorized, httpresponse.ErrAuthInvalidToken, "authentication required")
		return
	}
	st, found := h.svc.PendingInvite(r.Context(), actor)
	if !found {
		httpresponse.OK(w, r, http.StatusOK, map[string]any{"call": nil})
		return
	}
	httpresponse.OK(w, r, http.StatusOK, map[string]any{"call": map[string]any{
		"callId":     st.ID,
		"callerId":   st.Caller,
		"callerName": st.CallerName,
		"chatId":     st.ChatID,
		"offer":      st.Offer,
	}})
}

func (h *Handler) reject(w http.ResponseWriter, r *http.Request) {
	actor, ok := h.actor(r)
	if !ok {
		httpresponse.Fail(w, r, http.StatusUnauthorized, httpresponse.ErrAuthInvalidToken, "authentication required")
		return
	}
	callID, err := uuid.Parse(chi.URLParam(r, "callID"))
	if err != nil {
		httpresponse.Fail(w, r, http.StatusBadRequest, httpresponse.ErrValidationFailed, "invalid call id")
		return
	}
	if err := h.svc.Reject(r.Context(), actor, callID); err != nil {
		httpresponse.Fail(w, r, http.StatusForbidden, httpresponse.ErrAccessDenied, "cannot reject this call")
		return
	}
	httpresponse.OK(w, r, http.StatusOK, map[string]bool{"rejected": true})
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
