package chatfolders

import (
	"context"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"kisy-backend/pkg/httpjson"
	"kisy-backend/pkg/httpresponse"
)

// Handler exposes folder and archive endpoints. RequireAuth is applied by
// the router; every handler scopes to the acting user.
type Handler struct {
	svc   *Service
	actor func(*http.Request) (Actor, bool)
}

func NewHandler(svc *Service, actor func(*http.Request) (Actor, bool)) *Handler {
	return &Handler{svc: svc, actor: actor}
}

// FolderRoutes registers folder CRUD under /folders.
func (h *Handler) FolderRoutes(r chi.Router) {
	r.Get("/", h.list)
	r.Post("/", h.create)
	r.Put("/order", h.reorder)
	r.Patch("/{folderID}", h.rename)
	r.Delete("/{folderID}", h.delete)
	r.Post("/{folderID}/items", h.addItem)
	r.Delete("/{folderID}/items", h.removeItem)
}

// ChatRoutes registers the per-chat archive endpoints under /chats.
func (h *Handler) ChatRoutes(r chi.Router) {
	r.Put("/{chatType}/{chatID}/archive", h.archive)
	r.Delete("/{chatType}/{chatID}/archive", h.unarchive)
}

// SettingsRoutes registers the archived-chats listing under /settings
// (mirrors /settings/mutes from stage G).
func (h *Handler) SettingsRoutes(r chi.Router) {
	r.Get("/archived", h.listArchived)
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	actor, ok := h.actor(r)
	if !ok {
		unauth(w, r)
		return
	}
	folders, err := h.svc.ListFolders(r.Context(), actor)
	if err != nil {
		fail(w, r, err)
		return
	}
	httpresponse.OK(w, r, http.StatusOK, map[string]any{"folders": folders})
}

type folderNameRequest struct {
	Name string `json:"name"`
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	actor, ok := h.actor(r)
	if !ok {
		unauth(w, r)
		return
	}
	var req folderNameRequest
	if err := httpjson.Decode(w, r, &req); err != nil {
		badRequest(w, r, "malformed JSON body")
		return
	}
	folder, err := h.svc.CreateFolder(r.Context(), actor, req.Name)
	if err != nil {
		fail(w, r, err)
		return
	}
	httpresponse.OK(w, r, http.StatusCreated, map[string]any{"folder": folder})
}

func (h *Handler) rename(w http.ResponseWriter, r *http.Request) {
	actor, ok := h.actor(r)
	if !ok {
		unauth(w, r)
		return
	}
	folderID, ok2 := parseFolderID(w, r)
	if !ok2 {
		return
	}
	var req folderNameRequest
	if err := httpjson.Decode(w, r, &req); err != nil {
		badRequest(w, r, "malformed JSON body")
		return
	}
	if err := h.svc.RenameFolder(r.Context(), actor, folderID, req.Name); err != nil {
		fail(w, r, err)
		return
	}
	httpresponse.OK(w, r, http.StatusOK, map[string]any{"ok": true})
}

func (h *Handler) delete(w http.ResponseWriter, r *http.Request) {
	actor, ok := h.actor(r)
	if !ok {
		unauth(w, r)
		return
	}
	folderID, ok2 := parseFolderID(w, r)
	if !ok2 {
		return
	}
	if err := h.svc.DeleteFolder(r.Context(), actor, folderID); err != nil {
		fail(w, r, err)
		return
	}
	httpresponse.OK(w, r, http.StatusOK, map[string]any{"ok": true})
}

type reorderRequest struct {
	FolderIDs []string `json:"folderIds"`
}

func (h *Handler) reorder(w http.ResponseWriter, r *http.Request) {
	actor, ok := h.actor(r)
	if !ok {
		unauth(w, r)
		return
	}
	var req reorderRequest
	if err := httpjson.Decode(w, r, &req); err != nil {
		badRequest(w, r, "malformed JSON body")
		return
	}
	ids := make([]uuid.UUID, 0, len(req.FolderIDs))
	for _, raw := range req.FolderIDs {
		id, err := uuid.Parse(raw)
		if err != nil {
			badRequest(w, r, "folderIds must be valid UUIDs")
			return
		}
		ids = append(ids, id)
	}
	if err := h.svc.ReorderFolders(r.Context(), actor, ids); err != nil {
		fail(w, r, err)
		return
	}
	httpresponse.OK(w, r, http.StatusOK, map[string]any{"ok": true})
}

type itemRequest struct {
	ChatType string `json:"chatType"`
	ChatID   string `json:"chatId"`
}

func (h *Handler) addItem(w http.ResponseWriter, r *http.Request) {
	h.mutateItem(w, r, h.svc.AddItem)
}

func (h *Handler) removeItem(w http.ResponseWriter, r *http.Request) {
	h.mutateItem(w, r, h.svc.RemoveItem)
}

func (h *Handler) mutateItem(w http.ResponseWriter, r *http.Request, op func(ctx context.Context, actor Actor, folderID uuid.UUID, item Item) error) {
	actor, ok := h.actor(r)
	if !ok {
		unauth(w, r)
		return
	}
	folderID, ok2 := parseFolderID(w, r)
	if !ok2 {
		return
	}
	var req itemRequest
	if err := httpjson.Decode(w, r, &req); err != nil {
		badRequest(w, r, "malformed JSON body")
		return
	}
	chatID, err := uuid.Parse(req.ChatID)
	if err != nil {
		badRequest(w, r, "chatId must be a valid UUID")
		return
	}
	if err := op(r.Context(), actor, folderID, Item{ChatType: req.ChatType, ChatID: chatID}); err != nil {
		fail(w, r, err)
		return
	}
	httpresponse.OK(w, r, http.StatusOK, map[string]any{"ok": true})
}

func (h *Handler) archive(w http.ResponseWriter, r *http.Request) {
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
	if err := h.svc.Archive(r.Context(), actor, chi.URLParam(r, "chatType"), chatID); err != nil {
		fail(w, r, err)
		return
	}
	httpresponse.OK(w, r, http.StatusOK, map[string]any{"archived": true})
}

func (h *Handler) unarchive(w http.ResponseWriter, r *http.Request) {
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
	if err := h.svc.Unarchive(r.Context(), actor, chi.URLParam(r, "chatType"), chatID); err != nil {
		fail(w, r, err)
		return
	}
	httpresponse.OK(w, r, http.StatusOK, map[string]any{"archived": false})
}

func (h *Handler) listArchived(w http.ResponseWriter, r *http.Request) {
	actor, ok := h.actor(r)
	if !ok {
		unauth(w, r)
		return
	}
	list, err := h.svc.ListArchived(r.Context(), actor)
	if err != nil {
		fail(w, r, err)
		return
	}
	httpresponse.OK(w, r, http.StatusOK, map[string]any{"archived": list})
}

// --- helpers ---

func parseFolderID(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	id, err := uuid.Parse(chi.URLParam(r, "folderID"))
	if err != nil {
		badRequest(w, r, "invalid folder id")
		return uuid.Nil, false
	}
	return id, true
}

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
