package chatmedia

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"kisy-backend/pkg/httpresponse"
	"kisy-backend/pkg/pagination"
)

// Handler serves GET /chats/media — mounted inside the /chats subtree by
// the router (query-parameter addressing, the same convention as
// GET /messages, because the /chats/{chatID} wildcard already owns that
// path segment). RequireAuth is applied by the router.
type Handler struct {
	svc   *Service
	actor func(*http.Request) (Actor, bool)
}

func NewHandler(svc *Service, actor func(*http.Request) (Actor, bool)) *Handler {
	return &Handler{svc: svc, actor: actor}
}

// Routes registers the media aggregation endpoint on the /chats subrouter.
func (h *Handler) Routes(r chi.Router) {
	r.Get("/media", h.list)
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	actor, ok := h.actor(r)
	if !ok {
		httpresponse.Fail(w, r, http.StatusUnauthorized, httpresponse.ErrAuthInvalidToken, "authentication required")
		return
	}
	q := r.URL.Query()
	chatType := q.Get("chatType")
	if chatType != "private" && chatType != "group" {
		httpresponse.Fail(w, r, http.StatusBadRequest, httpresponse.ErrValidationFailed, "chatType must be 'private' or 'group'")
		return
	}
	chatID, err := uuid.Parse(q.Get("chatId"))
	if err != nil {
		httpresponse.Fail(w, r, http.StatusNotFound, httpresponse.ErrResourceNotFound, "chat not found")
		return
	}
	limit, _ := strconv.Atoi(q.Get("limit"))
	cursor := q.Get("cursor")

	kind := q.Get("kind")
	var payload any
	switch kind {
	case TabLinks:
		payload, err = h.svc.ListLinks(r.Context(), chatType, chatID, cursor, limit, actor)
	case TabMedia, TabFiles:
		payload, err = h.svc.ListAttachments(r.Context(), chatType, chatID, kind, cursor, limit, actor)
	default:
		httpresponse.Fail(w, r, http.StatusBadRequest, httpresponse.ErrValidationFailed, "kind must be media, files or links")
		return
	}
	if err != nil {
		writeError(w, r, err)
		return
	}
	httpresponse.OK(w, r, http.StatusOK, payload)
}

func writeError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, ErrBadKind), errors.Is(err, pagination.ErrInvalidCursor):
		httpresponse.Fail(w, r, http.StatusBadRequest, httpresponse.ErrValidationFailed, "invalid request")
	case errors.Is(err, ErrForbidden):
		httpresponse.Fail(w, r, http.StatusForbidden, httpresponse.ErrAccessDenied, "not permitted")
	default:
		// Chat authorizers surface their own not-found errors for hidden
		// chats; keep masking so existence never leaks.
		httpresponse.Fail(w, r, http.StatusNotFound, httpresponse.ErrResourceNotFound, "chat not found")
	}
}
