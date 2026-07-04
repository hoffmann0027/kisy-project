package attachments

import (
	"errors"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"kisy-backend/pkg/httpresponse"
)

// Actor is the acting user for authorization.
type Actor struct {
	UserID    uuid.UUID
	RoleLevel int
}

// Handler exposes attachment upload and download.
type Handler struct {
	svc   *Service
	actor func(*http.Request) (Actor, bool)
}

func NewHandler(svc *Service, actor func(*http.Request) (Actor, bool)) *Handler {
	return &Handler{svc: svc, actor: actor}
}

func (h *Handler) Routes(r chi.Router) {
	r.Post("/attachments", h.upload)
	r.Get("/attachments/{id}", h.serve)
}

// upload accepts a raw file body; the client sends the display name in the
// X-File-Name header (URL-encoded) and the bytes as the body.
func (h *Handler) upload(w http.ResponseWriter, r *http.Request) {
	actor, ok := h.actor(r)
	if !ok {
		httpresponse.Fail(w, r, http.StatusUnauthorized, httpresponse.ErrAuthInvalidToken, "authentication required")
		return
	}
	raw, err := io.ReadAll(http.MaxBytesReader(w, r.Body, MaxBytes+1024))
	if err != nil {
		httpresponse.Fail(w, r, http.StatusRequestEntityTooLarge, httpresponse.ErrValidationFailed, "file too large")
		return
	}
	name := r.Header.Get("X-File-Name")
	if decoded, derr := url.QueryUnescape(name); derr == nil {
		name = decoded
	}
	dto, err := h.svc.Upload(r.Context(), name, raw, actor.UserID)
	switch {
	case errors.Is(err, ErrTooLarge):
		httpresponse.Fail(w, r, http.StatusRequestEntityTooLarge, httpresponse.ErrValidationFailed, "file too large")
	case errors.Is(err, ErrEmpty):
		httpresponse.Fail(w, r, http.StatusBadRequest, httpresponse.ErrValidationFailed, "empty file")
	case err != nil:
		httpresponse.Fail(w, r, http.StatusInternalServerError, httpresponse.ErrInternal, "upload failed")
	default:
		httpresponse.OK(w, r, http.StatusCreated, map[string]any{"attachment": dto})
	}
}

func (h *Handler) serve(w http.ResponseWriter, r *http.Request) {
	actor, ok := h.actor(r)
	if !ok {
		httpresponse.Fail(w, r, http.StatusUnauthorized, httpresponse.ErrAuthInvalidToken, "authentication required")
		return
	}
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httpresponse.Fail(w, r, http.StatusNotFound, httpresponse.ErrResourceNotFound, "attachment not found")
		return
	}
	b, err := h.svc.Fetch(r.Context(), id, actor.UserID, actor.RoleLevel)
	if err != nil {
		httpresponse.Fail(w, r, http.StatusNotFound, httpresponse.ErrResourceNotFound, "attachment not found")
		return
	}

	w.Header().Set("Content-Type", b.MimeType)
	w.Header().Set("Content-Length", strconv.Itoa(len(b.Data)))
	w.Header().Set("Cache-Control", "private, max-age=86400")
	// Images render inline; everything else downloads. nosniff (set globally)
	// prevents a non-image being interpreted as active content.
	disp := "attachment"
	if strings.HasPrefix(b.MimeType, "image/") {
		disp = "inline"
	}
	w.Header().Set("Content-Disposition", disp+`; filename*=UTF-8''`+url.PathEscape(b.FileName))
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(b.Data)
}
