package notes

import (
	"errors"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"kisy-backend/pkg/httpjson"
	"kisy-backend/pkg/httpresponse"
)

// Handler exposes the personal notes endpoints. RequireAuth is applied by the
// router; every handler resolves the acting user and scopes to their notes.
type Handler struct {
	svc  *Service
	user func(*http.Request) (uuid.UUID, bool)
}

func NewHandler(svc *Service, user func(*http.Request) (uuid.UUID, bool)) *Handler {
	return &Handler{svc: svc, user: user}
}

func (h *Handler) Routes(r chi.Router) {
	r.Get("/", h.list)
	r.Post("/", h.createText)
	r.Post("/file", h.createFile)
	r.Get("/{id}/file", h.serveFile)
	r.Delete("/{id}", h.delete)
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.user(r)
	if !ok {
		unauth(w, r)
		return
	}
	items, err := h.svc.List(r.Context(), userID)
	if err != nil {
		fail(w, r, err)
		return
	}
	httpresponse.OK(w, r, http.StatusOK, map[string]any{"notes": items})
}

type createTextRequest struct {
	Text string `json:"text"`
}

func (h *Handler) createText(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.user(r)
	if !ok {
		unauth(w, r)
		return
	}
	var req createTextRequest
	if err := httpjson.Decode(w, r, &req); err != nil {
		httpresponse.Fail(w, r, http.StatusBadRequest, httpresponse.ErrValidationFailed, "malformed JSON body")
		return
	}
	note, err := h.svc.CreateText(r.Context(), userID, req.Text)
	if err != nil {
		fail(w, r, err)
		return
	}
	httpresponse.OK(w, r, http.StatusCreated, map[string]any{"note": note})
}

// createFile accepts a raw file body; the display name arrives in the
// X-File-Name header and an optional caption in X-Note-Text (both URL-encoded).
func (h *Handler) createFile(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.user(r)
	if !ok {
		unauth(w, r)
		return
	}
	raw, err := io.ReadAll(http.MaxBytesReader(w, r.Body, MaxFileBytes+1024))
	if err != nil {
		httpresponse.Fail(w, r, http.StatusRequestEntityTooLarge, httpresponse.ErrValidationFailed, "file too large")
		return
	}
	note, err := h.svc.CreateFile(r.Context(), userID, decodeHeader(r, "X-File-Name"), decodeHeader(r, "X-Note-Text"), raw)
	if err != nil {
		fail(w, r, err)
		return
	}
	httpresponse.OK(w, r, http.StatusCreated, map[string]any{"note": note})
}

func (h *Handler) serveFile(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.user(r)
	if !ok {
		unauth(w, r)
		return
	}
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httpresponse.Fail(w, r, http.StatusNotFound, httpresponse.ErrResourceNotFound, "note not found")
		return
	}
	b, err := h.svc.FetchFile(r.Context(), id, userID)
	if err != nil {
		httpresponse.Fail(w, r, http.StatusNotFound, httpresponse.ErrResourceNotFound, "note not found")
		return
	}
	w.Header().Set("Content-Type", b.FileType)
	w.Header().Set("Content-Length", strconv.Itoa(len(b.Data)))
	w.Header().Set("Cache-Control", "private, max-age=86400")
	disp := "attachment"
	if strings.HasPrefix(b.FileType, "image/") {
		disp = "inline"
	}
	w.Header().Set("Content-Disposition", disp+`; filename*=UTF-8''`+url.PathEscape(b.FileName))
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(b.Data)
}

func (h *Handler) delete(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.user(r)
	if !ok {
		unauth(w, r)
		return
	}
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httpresponse.Fail(w, r, http.StatusNotFound, httpresponse.ErrResourceNotFound, "note not found")
		return
	}
	if err := h.svc.Delete(r.Context(), id, userID); err != nil {
		fail(w, r, err)
		return
	}
	httpresponse.OK(w, r, http.StatusOK, map[string]any{"deleted": true})
}

// --- helpers ---

func decodeHeader(r *http.Request, name string) string {
	v := r.Header.Get(name)
	if decoded, err := url.QueryUnescape(v); err == nil {
		return decoded
	}
	return v
}

func unauth(w http.ResponseWriter, r *http.Request) {
	httpresponse.Fail(w, r, http.StatusUnauthorized, httpresponse.ErrAuthInvalidToken, "authentication required")
}

func fail(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, ErrNotFound):
		httpresponse.Fail(w, r, http.StatusNotFound, httpresponse.ErrResourceNotFound, "not found")
	case errors.Is(err, ErrForbidden):
		httpresponse.Fail(w, r, http.StatusForbidden, httpresponse.ErrAccessDenied, "not permitted")
	case errors.Is(err, ErrEmpty), errors.Is(err, ErrTooLong):
		httpresponse.Fail(w, r, http.StatusBadRequest, httpresponse.ErrValidationFailed, "invalid note")
	case errors.Is(err, ErrTooLarge):
		httpresponse.Fail(w, r, http.StatusRequestEntityTooLarge, httpresponse.ErrValidationFailed, "file too large")
	default:
		httpresponse.Fail(w, r, http.StatusInternalServerError, httpresponse.ErrInternal, "internal error")
	}
}
