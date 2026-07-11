package notifprefs

import (
	"errors"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"kisy-backend/pkg/httpjson"
	"kisy-backend/pkg/httpresponse"
)

// Handler exposes mute and notification-settings endpoints. RequireAuth is
// applied by the router; every handler scopes to the acting user.
type Handler struct {
	svc  *Service
	user func(*http.Request) (uuid.UUID, bool)
}

func NewHandler(svc *Service, user func(*http.Request) (uuid.UUID, bool)) *Handler {
	return &Handler{svc: svc, user: user}
}

// ChatRoutes registers the per-chat mute endpoints under /chats.
func (h *Handler) ChatRoutes(r chi.Router) {
	r.Put("/{chatType}/{chatID}/mute", h.mute)
	r.Delete("/{chatType}/{chatID}/mute", h.unmute)
}

// SettingsRoutes registers the notification-settings endpoints under /settings.
func (h *Handler) SettingsRoutes(r chi.Router) {
	r.Get("/notifications", h.getSettings)
	r.Put("/notifications", h.updateSettings)
	r.Get("/mutes", h.listMutes)
}

func (h *Handler) listMutes(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.user(r)
	if !ok {
		unauth(w, r)
		return
	}
	list, err := h.svc.ListMutes(r.Context(), userID)
	if err != nil {
		fail(w, r, err)
		return
	}
	httpresponse.OK(w, r, http.StatusOK, map[string]any{"mutes": list})
}

type muteRequest struct {
	// UntilSeconds is a duration from now in seconds; 0/absent = forever.
	UntilSeconds int64 `json:"untilSeconds"`
}

func (h *Handler) mute(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.user(r)
	if !ok {
		unauth(w, r)
		return
	}
	chatType := chi.URLParam(r, "chatType")
	chatID, err := uuid.Parse(chi.URLParam(r, "chatID"))
	if err != nil {
		badRequest(w, r, "invalid chat id")
		return
	}
	var req muteRequest
	// Body is optional; ignore decode errors for an empty body.
	_ = httpjson.Decode(w, r, &req)

	var until *time.Time
	if req.UntilSeconds > 0 {
		t := time.Now().Add(time.Duration(req.UntilSeconds) * time.Second)
		until = &t
	}
	if err := h.svc.Mute(r.Context(), userID, chatType, chatID, until); err != nil {
		fail(w, r, err)
		return
	}
	httpresponse.OK(w, r, http.StatusOK, map[string]any{"muted": true, "mutedUntil": until})
}

func (h *Handler) unmute(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.user(r)
	if !ok {
		unauth(w, r)
		return
	}
	chatID, err := uuid.Parse(chi.URLParam(r, "chatID"))
	if err != nil {
		badRequest(w, r, "invalid chat id")
		return
	}
	if err := h.svc.Unmute(r.Context(), userID, chi.URLParam(r, "chatType"), chatID); err != nil {
		fail(w, r, err)
		return
	}
	httpresponse.OK(w, r, http.StatusOK, map[string]any{"muted": false})
}

func (h *Handler) getSettings(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.user(r)
	if !ok {
		unauth(w, r)
		return
	}
	s, err := h.svc.GetSettings(r.Context(), userID)
	if err != nil {
		fail(w, r, err)
		return
	}
	httpresponse.OK(w, r, http.StatusOK, map[string]any{"settings": s})
}

func (h *Handler) updateSettings(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.user(r)
	if !ok {
		unauth(w, r)
		return
	}
	var req Settings
	if err := httpjson.Decode(w, r, &req); err != nil {
		badRequest(w, r, "malformed JSON body")
		return
	}
	s, err := h.svc.UpdateSettings(r.Context(), userID, req)
	if err != nil {
		fail(w, r, err)
		return
	}
	httpresponse.OK(w, r, http.StatusOK, map[string]any{"settings": s})
}

// --- helpers ---

func unauth(w http.ResponseWriter, r *http.Request) {
	httpresponse.Fail(w, r, http.StatusUnauthorized, httpresponse.ErrAuthInvalidToken, "authentication required")
}

func badRequest(w http.ResponseWriter, r *http.Request, msg string) {
	httpresponse.Fail(w, r, http.StatusBadRequest, httpresponse.ErrValidationFailed, msg)
}

func fail(w http.ResponseWriter, r *http.Request, err error) {
	if errors.Is(err, ErrValidation) {
		badRequest(w, r, "invalid request")
		return
	}
	httpresponse.Fail(w, r, http.StatusInternalServerError, httpresponse.ErrInternal, "internal error")
}
