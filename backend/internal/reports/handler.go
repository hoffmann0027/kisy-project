package reports

import (
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"kisy-backend/pkg/httpjson"
	"kisy-backend/pkg/httpresponse"
)

// Handler exposes reporting to everyone, and the queue to the CEO.
type Handler struct {
	svc   *Service
	actor func(*http.Request) (ActorMeta, bool)
}

func NewHandler(svc *Service, actor func(*http.Request) (ActorMeta, bool)) *Handler {
	return &Handler{svc: svc, actor: actor}
}

// Routes: what any signed-in person may do.
func (h *Handler) Routes(r chi.Router) {
	r.Post("/reports", h.create)
}

// AdminRoutes: the queue. Registered through admin.Mount, so it sits behind
// the same CEO gates as the rest of /admin.
func (h *Handler) AdminRoutes(r chi.Router) {
	r.Get("/reports", h.list)
	r.Get("/reports/summary", h.summary)
	r.Post("/reports/{reportID}/resolve", h.resolve)
}

type createRequest struct {
	TargetKind string `json:"targetKind"`
	TargetID   string `json:"targetId"`
	Reason     string `json:"reason"`
	Comment    string `json:"comment"`
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	actor, ok := h.actor(r)
	if !ok {
		unauthorized(w, r)
		return
	}
	var req createRequest
	if err := httpjson.Decode(w, r, &req); err != nil {
		httpresponse.Fail(w, r, http.StatusBadRequest, httpresponse.ErrValidationFailed, "malformed JSON body")
		return
	}
	targetID, err := uuid.Parse(req.TargetID)
	if err != nil {
		httpresponse.Fail(w, r, http.StatusBadRequest, httpresponse.ErrValidationFailed, "targetId must be a valid UUID")
		return
	}

	id, err := h.svc.Create(r.Context(), actor, Input{
		TargetKind: req.TargetKind,
		TargetID:   targetID,
		Reason:     req.Reason,
		Comment:    req.Comment,
	})
	switch {
	case errors.Is(err, ErrBadTarget), errors.Is(err, ErrBadReason):
		httpresponse.Fail(w, r, http.StatusBadRequest, httpresponse.ErrValidationFailed, "unknown target or reason")
	case errors.Is(err, ErrSelf):
		httpresponse.Fail(w, r, http.StatusBadRequest, httpresponse.ErrValidationFailed, "нельзя пожаловаться на себя")
	case errors.Is(err, ErrNotFound):
		// Nothing there — or nothing this account may see. Deliberately one
		// answer for both.
		httpresponse.Fail(w, r, http.StatusNotFound, httpresponse.ErrResourceNotFound, "not found")
	case err != nil:
		httpresponse.Fail(w, r, http.StatusInternalServerError, httpresponse.ErrInternal, "internal error")
	default:
		// Deliberately says nothing about what happens next: a reporter should
		// not learn whether this was the fifth report on a post.
		httpresponse.OK(w, r, http.StatusCreated, map[string]any{"reportId": id, "accepted": true})
	}
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	if _, ok := h.actor(r); !ok {
		unauthorized(w, r)
		return
	}
	list, err := h.svc.List(r.Context(), r.URL.Query().Get("status"), 0)
	if err != nil {
		httpresponse.Fail(w, r, http.StatusInternalServerError, httpresponse.ErrInternal, "internal error")
		return
	}
	httpresponse.OK(w, r, http.StatusOK, map[string]any{"reports": list})
}

func (h *Handler) summary(w http.ResponseWriter, r *http.Request) {
	if _, ok := h.actor(r); !ok {
		unauthorized(w, r)
		return
	}
	counts, err := h.svc.Summary(r.Context())
	if err != nil {
		httpresponse.Fail(w, r, http.StatusInternalServerError, httpresponse.ErrInternal, "internal error")
		return
	}
	httpresponse.OK(w, r, http.StatusOK, map[string]any{"counts": counts})
}

type resolveRequest struct {
	// Rejected: there was nothing to act on.
	Rejected bool `json:"rejected"`
}

func (h *Handler) resolve(w http.ResponseWriter, r *http.Request) {
	actor, ok := h.actor(r)
	if !ok {
		unauthorized(w, r)
		return
	}
	id, err := uuid.Parse(chi.URLParam(r, "reportID"))
	if err != nil {
		httpresponse.Fail(w, r, http.StatusBadRequest, httpresponse.ErrValidationFailed, "reportID must be a valid UUID")
		return
	}
	var req resolveRequest
	if err := httpjson.Decode(w, r, &req); err != nil {
		httpresponse.Fail(w, r, http.StatusBadRequest, httpresponse.ErrValidationFailed, "malformed JSON body")
		return
	}

	switch err := h.svc.Resolve(r.Context(), actor, id, req.Rejected); {
	case errors.Is(err, ErrNotFound):
		httpresponse.Fail(w, r, http.StatusNotFound, httpresponse.ErrResourceNotFound, "report not found")
	case err != nil:
		httpresponse.Fail(w, r, http.StatusInternalServerError, httpresponse.ErrInternal, "internal error")
	default:
		httpresponse.OK(w, r, http.StatusOK, map[string]any{"resolved": true})
	}
}

func unauthorized(w http.ResponseWriter, r *http.Request) {
	httpresponse.Fail(w, r, http.StatusUnauthorized, httpresponse.ErrAuthInvalidToken, "authentication required")
}
