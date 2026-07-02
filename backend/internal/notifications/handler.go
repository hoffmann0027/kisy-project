package notifications

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"kisy-backend/pkg/httpjson"
	"kisy-backend/pkg/httpresponse"
)

// Handler exposes /notifications. RequireAuth is applied by the router.
type Handler struct {
	svc    *Service
	userID func(*http.Request) (uuid.UUID, bool)
}

func NewHandler(svc *Service, userID func(*http.Request) (uuid.UUID, bool)) *Handler {
	return &Handler{svc: svc, userID: userID}
}

func (h *Handler) Routes(r chi.Router) {
	r.Get("/", h.list)
	r.Post("/read", h.markRead)
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	uid, ok := h.userID(r)
	if !ok {
		httpresponse.Fail(w, r, http.StatusUnauthorized, httpresponse.ErrAuthInvalidToken, "authentication required")
		return
	}
	limit := 0
	if raw := r.URL.Query().Get("limit"); raw != "" {
		limit, _ = strconv.Atoi(raw)
	}

	list, unread, err := h.svc.List(r.Context(), uid, limit)
	if err != nil {
		httpresponse.Fail(w, r, http.StatusInternalServerError, httpresponse.ErrInternal, "failed to list notifications")
		return
	}
	httpresponse.OK(w, r, http.StatusOK, map[string]any{"notifications": list, "unreadCount": unread})
}

type markReadRequest struct {
	// ID marks one notification read; when omitted, all are marked read.
	ID *string `json:"id"`
}

func (h *Handler) markRead(w http.ResponseWriter, r *http.Request) {
	uid, ok := h.userID(r)
	if !ok {
		httpresponse.Fail(w, r, http.StatusUnauthorized, httpresponse.ErrAuthInvalidToken, "authentication required")
		return
	}
	var req markReadRequest
	if err := httpjson.Decode(w, r, &req); err != nil {
		httpresponse.Fail(w, r, http.StatusBadRequest, httpresponse.ErrValidationFailed, "malformed JSON body")
		return
	}

	if req.ID == nil {
		if err := h.svc.MarkAllRead(r.Context(), uid); err != nil {
			httpresponse.Fail(w, r, http.StatusInternalServerError, httpresponse.ErrInternal, "internal error")
			return
		}
		httpresponse.OK(w, r, http.StatusOK, map[string]any{"ok": true})
		return
	}

	id, err := uuid.Parse(*req.ID)
	if err != nil {
		httpresponse.Fail(w, r, http.StatusBadRequest, httpresponse.ErrValidationFailed, "id must be a valid UUID")
		return
	}
	if err := h.svc.MarkRead(r.Context(), uid, id); err != nil {
		httpresponse.Fail(w, r, http.StatusInternalServerError, httpresponse.ErrInternal, "internal error")
		return
	}
	httpresponse.OK(w, r, http.StatusOK, map[string]any{"ok": true})
}
