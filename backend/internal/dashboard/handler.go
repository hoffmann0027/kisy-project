package dashboard

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"kisy-backend/pkg/httpresponse"
)

// Handler exposes GET /admin/dashboard. Registered through admin.Mount, so it
// sits behind the same CEO gates as the rest of /admin.
type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

// AdminRoutes registers the overview under /admin.
func (h *Handler) AdminRoutes(r chi.Router) {
	r.Get("/dashboard", h.overview)
}

func (h *Handler) overview(w http.ResponseWriter, r *http.Request) {
	o, err := h.svc.Overview(r.Context())
	if err != nil {
		httpresponse.Fail(w, r, http.StatusInternalServerError, httpresponse.ErrInternal, "failed to build the overview")
		return
	}
	httpresponse.OK(w, r, http.StatusOK, o)
}
