package chats

import (
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"kisy-backend/pkg/httpjson"
	"kisy-backend/pkg/httpresponse"
)

// Handler exposes /chats endpoints. RequireAuth is applied by the router.
type Handler struct {
	svc   *Service
	actor func(*http.Request) (ActorMeta, bool)
}

func NewHandler(svc *Service, actor func(*http.Request) (ActorMeta, bool)) *Handler {
	return &Handler{svc: svc, actor: actor}
}

func (h *Handler) Routes(r chi.Router) {
	r.Get("/", h.list)
	r.Post("/", h.open)
	r.Get("/{chatID}", h.get)
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	actor, ok := h.actor(r)
	if !ok {
		httpresponse.Fail(w, r, http.StatusUnauthorized, httpresponse.ErrAuthInvalidToken, "authentication required")
		return
	}

	dtos, err := h.svc.ListDTOsForUser(r.Context(), actor)
	if err != nil {
		httpresponse.Fail(w, r, http.StatusInternalServerError, httpresponse.ErrInternal, "failed to list chats")
		return
	}
	httpresponse.OK(w, r, http.StatusOK, map[string]any{"chats": dtos})
}

type openRequest struct {
	UserID string `json:"userId"`
}

func (h *Handler) open(w http.ResponseWriter, r *http.Request) {
	actor, ok := h.actor(r)
	if !ok {
		httpresponse.Fail(w, r, http.StatusUnauthorized, httpresponse.ErrAuthInvalidToken, "authentication required")
		return
	}

	var req openRequest
	if err := httpjson.Decode(w, r, &req); err != nil {
		httpresponse.Fail(w, r, http.StatusBadRequest, httpresponse.ErrValidationFailed, "malformed JSON body")
		return
	}
	targetID, err := uuid.Parse(req.UserID)
	if err != nil {
		httpresponse.Fail(w, r, http.StatusBadRequest, httpresponse.ErrValidationFailed, "userId must be a valid UUID")
		return
	}

	chat, err := h.svc.OpenPrivateChat(r.Context(), targetID, actor)
	switch {
	case errors.Is(err, ErrSelfChat):
		httpresponse.Fail(w, r, http.StatusBadRequest, httpresponse.ErrValidationFailed, "cannot open a chat with yourself")
	case errors.Is(err, ErrNotFound):
		httpresponse.Fail(w, r, http.StatusNotFound, httpresponse.ErrResourceNotFound, "user not found")
	case errors.Is(err, ErrCannotInitiate):
		// The target is real but out of reach; deny without confirming
		// their existence beyond what the actor already supplied.
		httpresponse.Fail(w, r, http.StatusForbidden, httpresponse.ErrAccessDenied, "you cannot initiate a conversation with this user")
	case err != nil:
		httpresponse.Fail(w, r, http.StatusInternalServerError, httpresponse.ErrInternal, "failed to open chat")
	default:
		httpresponse.OK(w, r, http.StatusOK, map[string]any{"chat": chat.ToDTO(actor.UserID)})
	}
}

func (h *Handler) get(w http.ResponseWriter, r *http.Request) {
	actor, ok := h.actor(r)
	if !ok {
		httpresponse.Fail(w, r, http.StatusUnauthorized, httpresponse.ErrAuthInvalidToken, "authentication required")
		return
	}
	id, err := uuid.Parse(chi.URLParam(r, "chatID"))
	if err != nil {
		httpresponse.Fail(w, r, http.StatusNotFound, httpresponse.ErrResourceNotFound, "chat not found")
		return
	}

	chat, err := h.svc.GetParticipating(r.Context(), id, actor)
	if errors.Is(err, ErrNotFound) {
		httpresponse.Fail(w, r, http.StatusNotFound, httpresponse.ErrResourceNotFound, "chat not found")
		return
	}
	if err != nil {
		httpresponse.Fail(w, r, http.StatusInternalServerError, httpresponse.ErrInternal, "internal error")
		return
	}

	httpresponse.OK(w, r, http.StatusOK, map[string]any{"chat": chat.ToDTO(actor.UserID)})
}
