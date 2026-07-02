package readstate

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"kisy-backend/pkg/httpjson"
	"kisy-backend/pkg/httpresponse"
)

// Handler exposes the mark-read endpoint. RequireAuth is applied by the
// router.
type Handler struct {
	svc   *Service
	actor func(*http.Request) (Actor, bool)
}

func NewHandler(svc *Service, actor func(*http.Request) (Actor, bool)) *Handler {
	return &Handler{svc: svc, actor: actor}
}

func (h *Handler) Routes(r chi.Router) {
	r.Post("/read", h.markRead)
}

type markReadRequest struct {
	ChatType  string `json:"chatType"`
	ChatID    string `json:"chatId"`
	MessageID string `json:"messageId"`
}

func (h *Handler) markRead(w http.ResponseWriter, r *http.Request) {
	actor, ok := h.actor(r)
	if !ok {
		httpresponse.Fail(w, r, http.StatusUnauthorized, httpresponse.ErrAuthInvalidToken, "authentication required")
		return
	}

	var req markReadRequest
	if err := httpjson.Decode(w, r, &req); err != nil {
		httpresponse.Fail(w, r, http.StatusBadRequest, httpresponse.ErrValidationFailed, "malformed JSON body")
		return
	}
	if req.ChatType != "private" && req.ChatType != "group" {
		httpresponse.Fail(w, r, http.StatusBadRequest, httpresponse.ErrValidationFailed, "chatType must be 'private' or 'group'")
		return
	}
	chatID, err := uuid.Parse(req.ChatID)
	if err != nil {
		httpresponse.Fail(w, r, http.StatusBadRequest, httpresponse.ErrValidationFailed, "chatId must be a valid UUID")
		return
	}
	messageID, err := uuid.Parse(req.MessageID)
	if err != nil {
		httpresponse.Fail(w, r, http.StatusBadRequest, httpresponse.ErrValidationFailed, "messageId must be a valid UUID")
		return
	}

	if err := h.svc.MarkRead(r.Context(), req.ChatType, chatID, messageID, actor); err != nil {
		// Authorization failures collapse to not-found to avoid leaking
		// the existence of chats the actor cannot access.
		httpresponse.Fail(w, r, http.StatusNotFound, httpresponse.ErrResourceNotFound, "chat not found")
		return
	}
	httpresponse.OK(w, r, http.StatusOK, map[string]any{"ok": true})
}
