package messages

import (
	"encoding/base64"
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"kisy-backend/pkg/httpjson"
	"kisy-backend/pkg/httpresponse"
	"kisy-backend/pkg/pagination"
)

// Handler exposes message endpoints. RequireAuth is applied by the router.
type Handler struct {
	svc   *Service
	actor func(*http.Request) (ActorMeta, bool)
}

func NewHandler(svc *Service, actor func(*http.Request) (ActorMeta, bool)) *Handler {
	return &Handler{svc: svc, actor: actor}
}

func (h *Handler) Routes(r chi.Router) {
	r.Post("/messages", h.send)
	r.Get("/messages", h.list)
	r.Get("/messages/pinned", h.listPinned)
	r.Patch("/messages/{messageID}", h.edit)
	r.Delete("/messages/{messageID}", h.delete)
	r.Post("/messages/{messageID}/pin", h.pin(true))
	r.Post("/messages/{messageID}/unpin", h.pin(false))
}

func (h *Handler) listPinned(w http.ResponseWriter, r *http.Request) {
	actor, ok := h.actor(r)
	if !ok {
		httpresponse.Fail(w, r, http.StatusUnauthorized, httpresponse.ErrAuthInvalidToken, "authentication required")
		return
	}
	q := r.URL.Query()
	chatType := q.Get("chatType")
	if chatType != ChatPrivate && chatType != ChatGroup {
		httpresponse.Fail(w, r, http.StatusBadRequest, httpresponse.ErrValidationFailed, "chatType must be 'private' or 'group'")
		return
	}
	chatID, err := uuid.Parse(q.Get("chatId"))
	if err != nil {
		httpresponse.Fail(w, r, http.StatusBadRequest, httpresponse.ErrValidationFailed, "chatId must be a valid UUID")
		return
	}
	list, err := h.svc.ListPinned(r.Context(), chatType, chatID, actor)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	if list == nil {
		list = []DTO{}
	}
	httpresponse.OK(w, r, http.StatusOK, map[string]any{"pinned": list})
}

func (h *Handler) pin(pin bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		actor, ok := h.actor(r)
		if !ok {
			httpresponse.Fail(w, r, http.StatusUnauthorized, httpresponse.ErrAuthInvalidToken, "authentication required")
			return
		}
		id, err := uuid.Parse(chi.URLParam(r, "messageID"))
		if err != nil {
			httpresponse.Fail(w, r, http.StatusNotFound, httpresponse.ErrResourceNotFound, "message not found")
			return
		}
		dto, err := h.svc.SetPinned(r.Context(), id, pin, actor)
		if err != nil {
			h.writeError(w, r, err)
			return
		}
		httpresponse.OK(w, r, http.StatusOK, map[string]any{"message": dto})
	}
}

type editRequest struct {
	Text string `json:"text"`
}

func (h *Handler) edit(w http.ResponseWriter, r *http.Request) {
	actor, ok := h.actor(r)
	if !ok {
		httpresponse.Fail(w, r, http.StatusUnauthorized, httpresponse.ErrAuthInvalidToken, "authentication required")
		return
	}
	id, err := uuid.Parse(chi.URLParam(r, "messageID"))
	if err != nil {
		httpresponse.Fail(w, r, http.StatusNotFound, httpresponse.ErrResourceNotFound, "message not found")
		return
	}
	var req editRequest
	if err := httpjson.Decode(w, r, &req); err != nil {
		httpresponse.Fail(w, r, http.StatusBadRequest, httpresponse.ErrValidationFailed, "malformed JSON body")
		return
	}
	dto, err := h.svc.Edit(r.Context(), id, req.Text, actor)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	httpresponse.OK(w, r, http.StatusOK, map[string]any{"message": dto})
}

type sendRequest struct {
	ChatType      string   `json:"chatType"`
	ChatID        string   `json:"chatId"`
	Text          string   `json:"text"`
	ReplyTo       *string  `json:"replyTo"`
	AttachmentIDs []string `json:"attachmentIds"`

	// E2EE body (base64 MLS ciphertext) — mutually exclusive with text.
	Ciphertext  string `json:"ciphertext"`
	Alg         *int16 `json:"alg"`
	Epoch       *int64 `json:"epoch"`
	ContentKind *int16 `json:"contentKind"`
}

func (h *Handler) send(w http.ResponseWriter, r *http.Request) {
	actor, ok := h.actor(r)
	if !ok {
		httpresponse.Fail(w, r, http.StatusUnauthorized, httpresponse.ErrAuthInvalidToken, "authentication required")
		return
	}

	var req sendRequest
	if err := httpjson.Decode(w, r, &req); err != nil {
		httpresponse.Fail(w, r, http.StatusBadRequest, httpresponse.ErrValidationFailed, "malformed JSON body")
		return
	}
	if req.ChatType != ChatPrivate && req.ChatType != ChatGroup {
		httpresponse.Fail(w, r, http.StatusBadRequest, httpresponse.ErrValidationFailed, "chatType must be 'private' or 'group'")
		return
	}
	chatID, err := uuid.Parse(req.ChatID)
	if err != nil {
		httpresponse.Fail(w, r, http.StatusBadRequest, httpresponse.ErrValidationFailed, "chatId must be a valid UUID")
		return
	}
	var replyTo *uuid.UUID
	if req.ReplyTo != nil {
		id, err := uuid.Parse(*req.ReplyTo)
		if err != nil {
			httpresponse.Fail(w, r, http.StatusBadRequest, httpresponse.ErrValidationFailed, "replyTo must be a valid UUID")
			return
		}
		replyTo = &id
	}
	attachmentIDs := make([]uuid.UUID, 0, len(req.AttachmentIDs))
	for _, raw := range req.AttachmentIDs {
		id, err := uuid.Parse(raw)
		if err != nil {
			httpresponse.Fail(w, r, http.StatusBadRequest, httpresponse.ErrValidationFailed, "attachmentIds must be valid UUIDs")
			return
		}
		attachmentIDs = append(attachmentIDs, id)
	}

	var ciphertext []byte
	if req.Ciphertext != "" {
		ciphertext, err = base64.StdEncoding.DecodeString(req.Ciphertext)
		if err != nil {
			httpresponse.Fail(w, r, http.StatusBadRequest, httpresponse.ErrValidationFailed, "ciphertext must be base64")
			return
		}
	}

	dto, err := h.svc.Send(r.Context(), SendInput{
		ChatType:      req.ChatType,
		ChatID:        chatID,
		Text:          req.Text,
		ReplyTo:       replyTo,
		AttachmentIDs: attachmentIDs,
		Ciphertext:    ciphertext,
		Alg:           req.Alg,
		Epoch:         req.Epoch,
		ContentKind:   req.ContentKind,
	}, actor)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	httpresponse.OK(w, r, http.StatusCreated, map[string]any{"message": dto})
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	actor, ok := h.actor(r)
	if !ok {
		httpresponse.Fail(w, r, http.StatusUnauthorized, httpresponse.ErrAuthInvalidToken, "authentication required")
		return
	}

	q := r.URL.Query()
	chatType := q.Get("chatType")
	if chatType != ChatPrivate && chatType != ChatGroup {
		httpresponse.Fail(w, r, http.StatusBadRequest, httpresponse.ErrValidationFailed, "chatType must be 'private' or 'group'")
		return
	}
	chatID, err := uuid.Parse(q.Get("chatId"))
	if err != nil {
		httpresponse.Fail(w, r, http.StatusBadRequest, httpresponse.ErrValidationFailed, "chatId must be a valid UUID")
		return
	}
	limit := 0
	if raw := q.Get("limit"); raw != "" {
		if limit, err = strconv.Atoi(raw); err != nil {
			httpresponse.Fail(w, r, http.StatusBadRequest, httpresponse.ErrValidationFailed, "limit must be an integer")
			return
		}
	}

	page, err := h.svc.List(r.Context(), chatType, chatID, q.Get("cursor"), limit, actor)
	if errors.Is(err, pagination.ErrInvalidCursor) {
		httpresponse.Fail(w, r, http.StatusBadRequest, httpresponse.ErrValidationFailed, "invalid cursor")
		return
	}
	if err != nil {
		h.writeError(w, r, err)
		return
	}

	httpresponse.OK(w, r, http.StatusOK, page)
}

func (h *Handler) delete(w http.ResponseWriter, r *http.Request) {
	actor, ok := h.actor(r)
	if !ok {
		httpresponse.Fail(w, r, http.StatusUnauthorized, httpresponse.ErrAuthInvalidToken, "authentication required")
		return
	}
	id, err := uuid.Parse(chi.URLParam(r, "messageID"))
	if err != nil {
		httpresponse.Fail(w, r, http.StatusNotFound, httpresponse.ErrResourceNotFound, "message not found")
		return
	}

	err = h.svc.Delete(r.Context(), id, actor)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	httpresponse.OK(w, r, http.StatusOK, map[string]any{"deleted": true})
}

func (h *Handler) writeResult(w http.ResponseWriter, r *http.Request, m *Message, err error, okStatus int) {
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	httpresponse.OK(w, r, okStatus, map[string]any{"message": m.ToDTO()})
}

// writeError maps domain errors to the API contract. Access failures and
// missing/hidden chats collapse to 404 so a caller cannot probe for the
// existence of resources above their clearance.
func (h *Handler) writeError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, ErrEmptyContent):
		httpresponse.Fail(w, r, http.StatusBadRequest, httpresponse.ErrValidationFailed, "message text must not be empty")
	case errors.Is(err, ErrBadChatType):
		httpresponse.Fail(w, r, http.StatusBadRequest, httpresponse.ErrValidationFailed, "unknown chat type")
	case errors.Is(err, ErrForbidden):
		httpresponse.Fail(w, r, http.StatusForbidden, httpresponse.ErrAccessDenied, "not permitted")
	case errors.Is(err, ErrNotFound):
		httpresponse.Fail(w, r, http.StatusNotFound, httpresponse.ErrResourceNotFound, "not found")
	default:
		httpresponse.Fail(w, r, http.StatusInternalServerError, httpresponse.ErrInternal, "internal error")
	}
}
