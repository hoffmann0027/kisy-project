package avatars

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"kisy-backend/pkg/httpresponse"
)

// Handler serves avatar images. Uploads are owned by the users and groups
// handlers (which enforce their own permissions); this only reads.
type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

// Routes mounts the avatar read endpoint at the /api/v1 root. It sits behind
// RequireAuth so avatars are only visible to authenticated users; browsers
// send the auth cookie automatically for <img> requests.
func (h *Handler) Routes(r chi.Router) {
	r.Get("/avatars/{ownerType}/{ownerID}", h.serve)
}

func (h *Handler) serve(w http.ResponseWriter, r *http.Request) {
	ownerType := chi.URLParam(r, "ownerType")
	if ownerType != OwnerUser && ownerType != OwnerGroup {
		httpresponse.Fail(w, r, http.StatusNotFound, httpresponse.ErrResourceNotFound, "avatar not found")
		return
	}
	ownerID, err := uuid.Parse(chi.URLParam(r, "ownerID"))
	if err != nil {
		httpresponse.Fail(w, r, http.StatusNotFound, httpresponse.ErrResourceNotFound, "avatar not found")
		return
	}

	img, err := h.svc.Load(r.Context(), ownerType, ownerID)
	if errors.Is(err, ErrNotFound) {
		httpresponse.Fail(w, r, http.StatusNotFound, httpresponse.ErrResourceNotFound, "avatar not found")
		return
	}
	if err != nil {
		httpresponse.Fail(w, r, http.StatusInternalServerError, httpresponse.ErrInternal, "internal error")
		return
	}

	w.Header().Set("Content-Type", img.ContentType)
	w.Header().Set("Content-Length", strconv.Itoa(len(img.Bytes)))
	// Cache-bustable via the ?v= query the avatar_url carries; the content at a
	// given version is immutable, so allow private caching.
	w.Header().Set("Cache-Control", "private, max-age=3600")
	w.Header().Set("ETag", strconv.FormatInt(img.UpdatedAt.UnixNano(), 10))
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(img.Bytes)
}
