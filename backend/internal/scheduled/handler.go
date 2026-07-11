package scheduled

import (
	"encoding/base64"
	"errors"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"kisy-backend/pkg/httpjson"
	"kisy-backend/pkg/httpresponse"
)

// Handler exposes the scheduled-messages endpoints. RequireAuth is applied
// by the router; every operation scopes to the acting user.
type Handler struct {
	svc   *Service
	actor func(*http.Request) (Actor, bool)
}

func NewHandler(svc *Service, actor func(*http.Request) (Actor, bool)) *Handler {
	return &Handler{svc: svc, actor: actor}
}

// Routes registers under the /api/v1 root, alongside /messages.
func (h *Handler) Routes(r chi.Router) {
	r.Post("/messages/schedule", h.schedule)
	r.Get("/messages/scheduled", h.list)
	r.Patch("/messages/scheduled/{id}", h.update)
	r.Delete("/messages/scheduled/{id}", h.cancel)
}

type scheduleRequest struct {
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

	SendAt time.Time `json:"sendAt"`
}

func (h *Handler) schedule(w http.ResponseWriter, r *http.Request) {
	actor, ok := h.actor(r)
	if !ok {
		unauth(w, r)
		return
	}
	var req scheduleRequest
	if err := httpjson.Decode(w, r, &req); err != nil {
		badRequest(w, r, "malformed JSON body")
		return
	}
	chatID, err := uuid.Parse(req.ChatID)
	if err != nil {
		badRequest(w, r, "chatId must be a valid UUID")
		return
	}
	var replyTo *uuid.UUID
	if req.ReplyTo != nil {
		id, err := uuid.Parse(*req.ReplyTo)
		if err != nil {
			badRequest(w, r, "replyTo must be a valid UUID")
			return
		}
		replyTo = &id
	}
	attachmentIDs := make([]uuid.UUID, 0, len(req.AttachmentIDs))
	for _, raw := range req.AttachmentIDs {
		id, err := uuid.Parse(raw)
		if err != nil {
			badRequest(w, r, "attachmentIds must be valid UUIDs")
			return
		}
		attachmentIDs = append(attachmentIDs, id)
	}
	ciphertext, ok2 := decodeCiphertext(w, r, req.Ciphertext)
	if !ok2 {
		return
	}

	dto, err := h.svc.Schedule(r.Context(), Input{
		ChatType: req.ChatType, ChatID: chatID,
		Text: req.Text, Ciphertext: ciphertext, Alg: req.Alg, Epoch: req.Epoch, ContentKind: req.ContentKind,
		ReplyTo: replyTo, AttachmentIDs: attachmentIDs, SendAt: req.SendAt,
	}, actor)
	if err != nil {
		fail(w, r, err)
		return
	}
	httpresponse.OK(w, r, http.StatusCreated, map[string]any{"scheduled": dto})
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	actor, ok := h.actor(r)
	if !ok {
		unauth(w, r)
		return
	}
	list, err := h.svc.List(r.Context(), actor)
	if err != nil {
		fail(w, r, err)
		return
	}
	if list == nil {
		list = []DTO{}
	}
	httpresponse.OK(w, r, http.StatusOK, map[string]any{"scheduled": list})
}

type updateRequest struct {
	Text        *string    `json:"text"`
	Ciphertext  string     `json:"ciphertext"`
	Alg         *int16     `json:"alg"`
	Epoch       *int64     `json:"epoch"`
	ContentKind *int16     `json:"contentKind"`
	SendAt      *time.Time `json:"sendAt"`
}

func (h *Handler) update(w http.ResponseWriter, r *http.Request) {
	actor, ok := h.actor(r)
	if !ok {
		unauth(w, r)
		return
	}
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		notFound(w, r)
		return
	}
	var req updateRequest
	if err := httpjson.Decode(w, r, &req); err != nil {
		badRequest(w, r, "malformed JSON body")
		return
	}
	ciphertext, ok2 := decodeCiphertext(w, r, req.Ciphertext)
	if !ok2 {
		return
	}
	dto, err := h.svc.Update(r.Context(), id, UpdateInput{
		Text: req.Text, Ciphertext: ciphertext, Alg: req.Alg, Epoch: req.Epoch, ContentKind: req.ContentKind,
		SendAt: req.SendAt,
	}, actor)
	if err != nil {
		fail(w, r, err)
		return
	}
	httpresponse.OK(w, r, http.StatusOK, map[string]any{"scheduled": dto})
}

func (h *Handler) cancel(w http.ResponseWriter, r *http.Request) {
	actor, ok := h.actor(r)
	if !ok {
		unauth(w, r)
		return
	}
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		notFound(w, r)
		return
	}
	if err := h.svc.Cancel(r.Context(), id, actor); err != nil {
		fail(w, r, err)
		return
	}
	httpresponse.OK(w, r, http.StatusOK, map[string]any{"canceled": true})
}

// --- helpers ---

func decodeCiphertext(w http.ResponseWriter, r *http.Request, raw string) ([]byte, bool) {
	if raw == "" {
		return nil, true
	}
	b, err := base64.StdEncoding.DecodeString(raw)
	if err != nil {
		badRequest(w, r, "ciphertext must be base64")
		return nil, false
	}
	return b, true
}

func unauth(w http.ResponseWriter, r *http.Request) {
	httpresponse.Fail(w, r, http.StatusUnauthorized, httpresponse.ErrAuthInvalidToken, "authentication required")
}

func badRequest(w http.ResponseWriter, r *http.Request, msg string) {
	httpresponse.Fail(w, r, http.StatusBadRequest, httpresponse.ErrValidationFailed, msg)
}

func notFound(w http.ResponseWriter, r *http.Request) {
	httpresponse.Fail(w, r, http.StatusNotFound, httpresponse.ErrResourceNotFound, "not found")
}

func fail(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, ErrValidation):
		badRequest(w, r, "invalid request")
	case errors.Is(err, ErrNotFound):
		notFound(w, r)
	default:
		httpresponse.Fail(w, r, http.StatusInternalServerError, httpresponse.ErrInternal, "internal error")
	}
}
