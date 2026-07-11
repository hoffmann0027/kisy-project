package linkpreview

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"kisy-backend/pkg/httpjson"
	"kisy-backend/pkg/httpresponse"
)

// Handler exposes link-preview fetching. Both endpoints require auth (applied
// by the router) — only signed-in users may drive the server's outbound
// fetcher, and both go through the SSRF guard.
type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

func (h *Handler) Routes(r chi.Router) {
	r.Post("/link-preview", h.preview)
	r.Get("/link-preview/image", h.image)
}

type previewRequest struct {
	URL string `json:"url"`
}

func (h *Handler) preview(w http.ResponseWriter, r *http.Request) {
	var req previewRequest
	if err := httpjson.Decode(w, r, &req); err != nil {
		httpresponse.Fail(w, r, http.StatusBadRequest, httpresponse.ErrValidationFailed, "malformed JSON body")
		return
	}
	p, err := h.svc.Preview(r.Context(), req.URL)
	if err != nil {
		writeError(w, r, err)
		return
	}
	httpresponse.OK(w, r, http.StatusOK, map[string]any{"preview": p})
}

func (h *Handler) image(w http.ResponseWriter, r *http.Request) {
	rawURL := r.URL.Query().Get("url")
	data, mime, err := h.svc.ImageProxy(r.Context(), rawURL)
	if err != nil {
		writeError(w, r, err)
		return
	}
	// Serve as an opaque download-safe image; nosniff (set globally) prevents
	// a mislabeled body being treated as active content.
	w.Header().Set("Content-Type", mime)
	w.Header().Set("Content-Length", strconv.Itoa(len(data)))
	w.Header().Set("Cache-Control", "private, max-age=21600")
	w.Header().Set("Content-Disposition", "inline")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(data)
}

func writeError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, ErrBlockedURL):
		httpresponse.Fail(w, r, http.StatusBadRequest, httpresponse.ErrValidationFailed, "url is not allowed")
	case errors.Is(err, ErrNotHTML), errors.Is(err, ErrTooLarge), errors.Is(err, ErrFetchFailed):
		// No usable preview — a 404 keeps the client's handling simple and
		// leaks nothing about why the fetch failed.
		httpresponse.Fail(w, r, http.StatusNotFound, httpresponse.ErrResourceNotFound, "no preview available")
	default:
		httpresponse.Fail(w, r, http.StatusNotFound, httpresponse.ErrResourceNotFound, "no preview available")
	}
}
