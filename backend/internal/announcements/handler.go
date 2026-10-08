package announcements

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"kisy-backend/pkg/httpjson"
	"kisy-backend/pkg/httpresponse"
)

// Handler exposes /announcements. RequireAuth is applied by the router; who
// may send is decided by the service from the caller's live level.
type Handler struct {
	svc   *Service
	actor func(*http.Request) (ActorMeta, bool)
}

func NewHandler(svc *Service, actor func(*http.Request) (ActorMeta, bool)) *Handler {
	return &Handler{svc: svc, actor: actor}
}

func (h *Handler) Routes(r chi.Router) {
	r.Get("/", h.list)
	r.Post("/", h.send)
	r.Delete("/{announcementID}", h.revoke)
}

type sendRequest struct {
	Audience string `json:"audience"`
	Levels   []int  `json:"levels"`
	UserID   string `json:"userId"`
	Title    string `json:"title"`
	Body     string `json:"body"`
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
	in := Input{Audience: req.Audience, Levels: req.Levels, Title: req.Title, Body: req.Body}
	if req.UserID != "" {
		id, err := uuid.Parse(req.UserID)
		if err != nil {
			httpresponse.Fail(w, r, http.StatusBadRequest, httpresponse.ErrValidationFailed, "userId must be a valid UUID")
			return
		}
		in.UserID = id
	}
	a, err := h.svc.Send(r.Context(), actor, in)
	if err != nil {
		writeError(w, r, err)
		return
	}
	httpresponse.OK(w, r, http.StatusCreated, map[string]any{"announcement": a})
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	actor, ok := h.actor(r)
	if !ok {
		httpresponse.Fail(w, r, http.StatusUnauthorized, httpresponse.ErrAuthInvalidToken, "authentication required")
		return
	}
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	items, err := h.svc.List(r.Context(), actor, limit)
	if err != nil {
		writeError(w, r, err)
		return
	}
	httpresponse.OK(w, r, http.StatusOK, map[string]any{"announcements": items})
}

func (h *Handler) revoke(w http.ResponseWriter, r *http.Request) {
	actor, ok := h.actor(r)
	if !ok {
		httpresponse.Fail(w, r, http.StatusUnauthorized, httpresponse.ErrAuthInvalidToken, "authentication required")
		return
	}
	id, err := uuid.Parse(chi.URLParam(r, "announcementID"))
	if err != nil {
		httpresponse.Fail(w, r, http.StatusBadRequest, httpresponse.ErrValidationFailed, "invalid announcement id")
		return
	}
	if err := h.svc.Revoke(r.Context(), actor, id); err != nil {
		writeError(w, r, err)
		return
	}
	httpresponse.OK(w, r, http.StatusOK, map[string]bool{"revoked": true})
}

func writeError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, ErrValidation):
		httpresponse.Fail(w, r, http.StatusBadRequest, httpresponse.ErrValidationFailed,
			fmt.Sprintf("audience must be all, basic, levels or user; title 1-%d and body 1-%d characters", MaxTitle, MaxBody))
	case errors.Is(err, ErrNotAuthor):
		httpresponse.Fail(w, r, http.StatusForbidden, httpresponse.ErrAccessDenied, "only levels 1-3 may send announcements")
	case errors.Is(err, ErrAboveYou):
		httpresponse.Fail(w, r, http.StatusForbidden, httpresponse.ErrAccessDenied, "cannot address a level above your own")
	case errors.Is(err, ErrUnreachable):
		httpresponse.Fail(w, r, http.StatusForbidden, httpresponse.ErrAccessDenied, "this person cannot be notified")
	case errors.Is(err, ErrQuota):
		// Written for the screen (QUOTA_EXCEEDED is a user-facing code).
		httpresponse.Fail(w, r, http.StatusTooManyRequests, httpresponse.ErrQuotaExceeded,
			fmt.Sprintf("Лимит на сутки исчерпан: не больше %d рассылок и %d личных уведомлений за 24 часа", BroadcastsPerDay, PersonalPerDay))
	case errors.Is(err, ErrNotFound):
		httpresponse.Fail(w, r, http.StatusNotFound, httpresponse.ErrResourceNotFound, "announcement not found")
	default:
		httpresponse.Fail(w, r, http.StatusInternalServerError, httpresponse.ErrInternal, "internal error")
	}
}
