package favorites

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"kisy-backend/pkg/httpjson"
	"kisy-backend/pkg/httpresponse"
)

// Handler exposes /favorites. RequireAuth is applied by the router.
type Handler struct {
	svc   *Service
	actor func(*http.Request) (Actor, bool)
}

func NewHandler(svc *Service, actor func(*http.Request) (Actor, bool)) *Handler {
	return &Handler{svc: svc, actor: actor}
}

func (h *Handler) Routes(r chi.Router) {
	r.Get("/", h.list)
	r.Put("/", h.set)
	r.Delete("/", h.remove)
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	actor, ok := h.actor(r)
	if !ok {
		httpresponse.Fail(w, r, http.StatusUnauthorized, httpresponse.ErrAuthInvalidToken, "authentication required")
		return
	}
	list, err := h.svc.List(r.Context(), actor)
	if err != nil {
		httpresponse.Fail(w, r, http.StatusInternalServerError, httpresponse.ErrInternal, "failed to list favorites")
		return
	}
	if list == nil {
		list = []Favorite{}
	}
	httpresponse.OK(w, r, http.StatusOK, map[string]any{"favorites": list})
}

type setRequest struct {
	ChatType    string `json:"chatType"`
	ChatID      string `json:"chatId"`
	IsPinned    bool   `json:"isPinned"`
	PinnedOrder *int   `json:"pinnedOrder"`
}

func (h *Handler) set(w http.ResponseWriter, r *http.Request) {
	actor, ok := h.actor(r)
	if !ok {
		httpresponse.Fail(w, r, http.StatusUnauthorized, httpresponse.ErrAuthInvalidToken, "authentication required")
		return
	}
	var req setRequest
	if err := httpjson.Decode(w, r, &req); err != nil {
		httpresponse.Fail(w, r, http.StatusBadRequest, httpresponse.ErrValidationFailed, "malformed JSON body")
		return
	}
	chatType, chatID, ok := parseChatRef(w, r, req.ChatType, req.ChatID)
	if !ok {
		return
	}

	err := h.svc.Set(r.Context(), Favorite{
		ChatType:    chatType,
		ChatID:      chatID,
		IsPinned:    req.IsPinned,
		PinnedOrder: req.PinnedOrder,
	}, actor)
	if err != nil {
		// Authorization failure is masked as not-found (hidden chat).
		httpresponse.Fail(w, r, http.StatusNotFound, httpresponse.ErrResourceNotFound, "chat not found")
		return
	}
	httpresponse.OK(w, r, http.StatusOK, map[string]any{"ok": true})
}

type removeRequest struct {
	ChatType string `json:"chatType"`
	ChatID   string `json:"chatId"`
}

func (h *Handler) remove(w http.ResponseWriter, r *http.Request) {
	actor, ok := h.actor(r)
	if !ok {
		httpresponse.Fail(w, r, http.StatusUnauthorized, httpresponse.ErrAuthInvalidToken, "authentication required")
		return
	}
	var req removeRequest
	if err := httpjson.Decode(w, r, &req); err != nil {
		httpresponse.Fail(w, r, http.StatusBadRequest, httpresponse.ErrValidationFailed, "malformed JSON body")
		return
	}
	chatType, chatID, ok := parseChatRef(w, r, req.ChatType, req.ChatID)
	if !ok {
		return
	}
	if err := h.svc.Remove(r.Context(), chatType, chatID, actor); err != nil {
		httpresponse.Fail(w, r, http.StatusInternalServerError, httpresponse.ErrInternal, "failed to remove favorite")
		return
	}
	httpresponse.OK(w, r, http.StatusOK, map[string]any{"ok": true})
}

func parseChatRef(w http.ResponseWriter, r *http.Request, chatType, chatID string) (string, uuid.UUID, bool) {
	if chatType != "private" && chatType != "group" {
		httpresponse.Fail(w, r, http.StatusBadRequest, httpresponse.ErrValidationFailed, "chatType must be 'private' or 'group'")
		return "", uuid.Nil, false
	}
	id, err := uuid.Parse(chatID)
	if err != nil {
		httpresponse.Fail(w, r, http.StatusBadRequest, httpresponse.ErrValidationFailed, "chatId must be a valid UUID")
		return "", uuid.Nil, false
	}
	return chatType, id, true
}
