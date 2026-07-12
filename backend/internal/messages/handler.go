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
	r.Post("/messages/forward", h.forward)
	r.Get("/messages", h.list)
	r.Get("/messages/thread", h.listThread)
	r.Get("/messages/pinned", h.listPinned)
	r.Patch("/messages/{messageID}", h.edit)
	r.Delete("/messages/{messageID}", h.delete)
	r.Post("/messages/{messageID}/pin", h.pin(true))
	r.Post("/messages/{messageID}/unpin", h.pin(false))
	r.Put("/messages/{messageID}/expiry", h.setExpiry)
}

type expiryRequest struct {
	// TTLSeconds counts from now; null/0 clears the timer.
	TTLSeconds *int64 `json:"ttlSeconds"`
}

func (h *Handler) setExpiry(w http.ResponseWriter, r *http.Request) {
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
	var req expiryRequest
	if err := httpjson.Decode(w, r, &req); err != nil {
		httpresponse.Fail(w, r, http.StatusBadRequest, httpresponse.ErrValidationFailed, "malformed JSON body")
		return
	}
	if req.TTLSeconds != nil && (*req.TTLSeconds < 0 || *req.TTLSeconds > maxTTLSeconds) {
		httpresponse.Fail(w, r, http.StatusBadRequest, httpresponse.ErrValidationFailed, "ttlSeconds out of range")
		return
	}
	dto, err := h.svc.SetExpiry(r.Context(), id, req.TTLSeconds, actor)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	httpresponse.OK(w, r, http.StatusOK, map[string]any{"message": dto})
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

	// Forwarded-from attribution for a client-side (E2EE) forward.
	ForwardedFromSenderID   *string `json:"forwardedFromSenderId"`
	ForwardedFromSenderName *string `json:"forwardedFromSenderName"`

	// Per-message disappearing timer in seconds (stage J, additive);
	// overrides the chat's default TTL for this message.
	TTLSeconds *int64 `json:"ttlSeconds"`

	// Thread reply (stage K, additive): root message id, groups only.
	ThreadRootID *string `json:"threadRootId"`
}

// maxTTLSeconds bounds a disappearing timer (1 year).
const maxTTLSeconds = 365 * 24 * 3600

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

	var fwdSenderID *uuid.UUID
	if req.ForwardedFromSenderID != nil {
		id, err := uuid.Parse(*req.ForwardedFromSenderID)
		if err != nil {
			httpresponse.Fail(w, r, http.StatusBadRequest, httpresponse.ErrValidationFailed, "forwardedFromSenderId must be a valid UUID")
			return
		}
		fwdSenderID = &id
	}

	if req.TTLSeconds != nil && (*req.TTLSeconds < 0 || *req.TTLSeconds > maxTTLSeconds) {
		httpresponse.Fail(w, r, http.StatusBadRequest, httpresponse.ErrValidationFailed, "ttlSeconds out of range")
		return
	}
	var threadRootID *uuid.UUID
	if req.ThreadRootID != nil {
		id, err := uuid.Parse(*req.ThreadRootID)
		if err != nil {
			httpresponse.Fail(w, r, http.StatusBadRequest, httpresponse.ErrValidationFailed, "threadRootId must be a valid UUID")
			return
		}
		threadRootID = &id
	}

	dto, err := h.svc.Send(r.Context(), SendInput{
		ChatType:                req.ChatType,
		ChatID:                  chatID,
		Text:                    req.Text,
		ReplyTo:                 replyTo,
		AttachmentIDs:           attachmentIDs,
		Ciphertext:              ciphertext,
		Alg:                     req.Alg,
		Epoch:                   req.Epoch,
		ContentKind:             req.ContentKind,
		ForwardedFromSenderID:   fwdSenderID,
		ForwardedFromSenderName: req.ForwardedFromSenderName,
		TTLSeconds:              req.TTLSeconds,
		ThreadRootID:            threadRootID,
	}, actor)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	httpresponse.OK(w, r, http.StatusCreated, map[string]any{"message": dto})
}

type forwardRequest struct {
	SourceMessageIDs []string `json:"sourceMessageIds"`
	TargetChatType   string   `json:"targetChatType"`
	TargetChatID     string   `json:"targetChatId"`
}

func (h *Handler) forward(w http.ResponseWriter, r *http.Request) {
	actor, ok := h.actor(r)
	if !ok {
		httpresponse.Fail(w, r, http.StatusUnauthorized, httpresponse.ErrAuthInvalidToken, "authentication required")
		return
	}
	var req forwardRequest
	if err := httpjson.Decode(w, r, &req); err != nil {
		httpresponse.Fail(w, r, http.StatusBadRequest, httpresponse.ErrValidationFailed, "malformed JSON body")
		return
	}
	if req.TargetChatType != ChatPrivate && req.TargetChatType != ChatGroup {
		httpresponse.Fail(w, r, http.StatusBadRequest, httpresponse.ErrValidationFailed, "targetChatType must be 'private' or 'group'")
		return
	}
	targetID, err := uuid.Parse(req.TargetChatID)
	if err != nil {
		httpresponse.Fail(w, r, http.StatusBadRequest, httpresponse.ErrValidationFailed, "targetChatId must be a valid UUID")
		return
	}
	if len(req.SourceMessageIDs) == 0 || len(req.SourceMessageIDs) > 50 {
		httpresponse.Fail(w, r, http.StatusBadRequest, httpresponse.ErrValidationFailed, "sourceMessageIds must contain 1..50 ids")
		return
	}
	ids := make([]uuid.UUID, 0, len(req.SourceMessageIDs))
	for _, raw := range req.SourceMessageIDs {
		id, err := uuid.Parse(raw)
		if err != nil {
			httpresponse.Fail(w, r, http.StatusBadRequest, httpresponse.ErrValidationFailed, "sourceMessageIds must be valid UUIDs")
			return
		}
		ids = append(ids, id)
	}

	dtos, err := h.svc.Forward(r.Context(), ForwardInput{
		SourceMessageIDs: ids,
		TargetChatType:   req.TargetChatType,
		TargetChatID:     targetID,
	}, actor)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	httpresponse.OK(w, r, http.StatusCreated, map[string]any{"messages": dtos})
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

// listThread pages one thread's replies (stage K). Access mirrors the main
// feed; an unknown or inaccessible root is a masked 404.
func (h *Handler) listThread(w http.ResponseWriter, r *http.Request) {
	actor, ok := h.actor(r)
	if !ok {
		httpresponse.Fail(w, r, http.StatusUnauthorized, httpresponse.ErrAuthInvalidToken, "authentication required")
		return
	}

	q := r.URL.Query()
	rootID, err := uuid.Parse(q.Get("rootId"))
	if err != nil {
		httpresponse.Fail(w, r, http.StatusBadRequest, httpresponse.ErrValidationFailed, "rootId must be a valid UUID")
		return
	}
	limit := 0
	if raw := q.Get("limit"); raw != "" {
		if limit, err = strconv.Atoi(raw); err != nil {
			httpresponse.Fail(w, r, http.StatusBadRequest, httpresponse.ErrValidationFailed, "limit must be an integer")
			return
		}
	}

	page, err := h.svc.ListThread(r.Context(), rootID, q.Get("cursor"), limit, actor)
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
	case errors.Is(err, ErrForwardBroadens):
		httpresponse.Fail(w, r, http.StatusForbidden, httpresponse.ErrAccessDenied, "cannot forward to a broader audience")
	case errors.Is(err, ErrForwardEncrypted):
		httpresponse.Fail(w, r, http.StatusConflict, httpresponse.ErrValidationFailed, "encrypted messages are forwarded from the app")
	case errors.Is(err, ErrForbidden):
		httpresponse.Fail(w, r, http.StatusForbidden, httpresponse.ErrAccessDenied, "not permitted")
	case errors.Is(err, ErrNotFound):
		httpresponse.Fail(w, r, http.StatusNotFound, httpresponse.ErrResourceNotFound, "not found")
	default:
		httpresponse.Fail(w, r, http.StatusInternalServerError, httpresponse.ErrInternal, "internal error")
	}
}
