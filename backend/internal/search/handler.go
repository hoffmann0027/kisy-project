package search

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"kisy-backend/pkg/httpresponse"
)

// Handler exposes GET /search. RequireAuth is applied by the router.
type Handler struct {
	svc    *Service
	userID func(*http.Request) (uuid.UUID, bool)
}

func NewHandler(svc *Service, userID func(*http.Request) (uuid.UUID, bool)) *Handler {
	return &Handler{svc: svc, userID: userID}
}

func (h *Handler) Routes(r chi.Router) {
	r.Get("/", h.search)
}

func (h *Handler) search(w http.ResponseWriter, r *http.Request) {
	uid, ok := h.userID(r)
	if !ok {
		httpresponse.Fail(w, r, http.StatusUnauthorized, httpresponse.ErrAuthInvalidToken, "authentication required")
		return
	}
	limit := 0
	if raw := r.URL.Query().Get("limit"); raw != "" {
		limit, _ = strconv.Atoi(raw)
	}
	results, err := h.svc.Search(r.Context(), uid, r.URL.Query().Get("q"), limit)
	if err != nil {
		httpresponse.Fail(w, r, http.StatusInternalServerError, httpresponse.ErrInternal, "search failed")
		return
	}
	httpresponse.OK(w, r, http.StatusOK, map[string]any{"results": results})
}
