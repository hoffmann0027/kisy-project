package invitations

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"kisy-backend/pkg/httpresponse"
)

// Handler exposes POST /invites. The main router wraps it with RequireAuth
// and RequireClearance(1): only the CEO can issue invitations.
type Handler struct {
	svc  *Service
	meta func(*http.Request) (CreatorMeta, bool)
}

func NewHandler(svc *Service, meta func(*http.Request) (CreatorMeta, bool)) *Handler {
	return &Handler{svc: svc, meta: meta}
}

func (h *Handler) Routes(r chi.Router) {
	r.Post("/", h.create)
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	meta, ok := h.meta(r)
	if !ok {
		httpresponse.Fail(w, r, http.StatusUnauthorized, httpresponse.ErrAuthInvalidToken, "authentication required")
		return
	}

	created, err := h.svc.Create(r.Context(), meta)
	if err != nil {
		httpresponse.Fail(w, r, http.StatusInternalServerError, httpresponse.ErrInternal, "failed to create invitation")
		return
	}

	httpresponse.OK(w, r, http.StatusCreated, created)
}
