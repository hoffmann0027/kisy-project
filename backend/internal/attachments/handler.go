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

	"kisy-backend/pkg/httpjson"
	"kisy-backend/pkg/httpresponse"
)

// Actor is the acting user for authorization.
type Actor struct {
	UserID    uuid.UUID
	RoleLevel int
}

// Handler exposes attachment upload and download: the single-shot POST for
// small files and the chunked init → chunk → complete flow for large ones.
type Handler struct {
	svc   *Service
	actor func(*http.Request) (Actor, bool)
}

func NewHandler(svc *Service, actor func(*http.Request) (Actor, bool)) *Handler {
	return &Handler{svc: svc, actor: actor}
}

func (h *Handler) Routes(r chi.Router) {
	r.Post("/attachments", h.upload)
	r.Get("/attachments/limit", h.limit)
	r.Post("/attachments/init", h.initUpload)
	r.Get("/attachments/{id}/upload-status", h.uploadStatus)
	r.Put("/attachments/{id}/chunk", h.putChunk)
	r.Post("/attachments/{id}/complete", h.completeUpload)
	r.Get("/attachments/{id}", h.serve)
}

// metaFromHeaders reads optional media metadata from X-Attachment-* headers
// (single-shot path keeps its raw-body contract; JSON meta lives in headers).
func metaFromHeaders(r *http.Request) (Meta, bool) {
	var meta Meta
	meta.Kind = r.Header.Get("X-Attachment-Kind")
	if v := r.Header.Get("X-Attachment-Duration-Ms"); v != "" {
		n, err := strconv.ParseInt(v, 10, 32)
		if err != nil {
			return Meta{}, false
		}
		d := int32(n)
		meta.DurationMs = &d
	}
	if v := r.Header.Get("X-Attachment-Width"); v != "" {
		n, err := strconv.ParseInt(v, 10, 32)
		if err != nil {
			return Meta{}, false
		}
		w := int32(n)
		meta.Width = &w
	}
	if v := r.Header.Get("X-Attachment-Height"); v != "" {
		n, err := strconv.ParseInt(v, 10, 32)
		if err != nil {
			return Meta{}, false
		}
		hh := int32(n)
		meta.Height = &hh
	}
	return meta, true
}

// upload accepts a raw file body; the client sends the display name in the
// X-File-Name header (URL-encoded) and the bytes as the body.
func (h *Handler) upload(w http.ResponseWriter, r *http.Request) {
	actor, ok := h.actor(r)
	if !ok {
		unauthorized(w, r)
		return
	}
	maxBytes, _ := h.svc.Limits(actor.RoleLevel)
	raw, err := io.ReadAll(http.MaxBytesReader(w, r.Body, maxBytes+1024))
	if err != nil {
		httpresponse.Fail(w, r, http.StatusRequestEntityTooLarge, httpresponse.ErrValidationFailed, "file too large")
		return
	}
	name := r.Header.Get("X-File-Name")
	if decoded, derr := url.QueryUnescape(name); derr == nil {
		name = decoded
	}
	meta, ok := metaFromHeaders(r)
	if !ok {
		httpresponse.Fail(w, r, http.StatusBadRequest, httpresponse.ErrValidationFailed, "invalid attachment metadata")
		return
	}
	dto, err := h.svc.Upload(r.Context(), name, raw, actor.UserID, actor.RoleLevel, meta)
	if err != nil {
		writeError(w, r, err)
		return
	}
	httpresponse.OK(w, r, http.StatusCreated, map[string]any{"attachment": dto})
}

// limit tells the client its upload policy so nothing is hardcoded in the UI.
func (h *Handler) limit(w http.ResponseWriter, r *http.Request) {
	actor, ok := h.actor(r)
	if !ok {
		unauthorized(w, r)
		return
	}
	maxBytes, chunkBytes := h.svc.Limits(actor.RoleLevel)
	httpresponse.OK(w, r, http.StatusOK, map[string]any{"maxBytes": maxBytes, "chunkBytes": chunkBytes})
}

type initUploadRequest struct {
	FileName  string `json:"fileName"`
	SizeBytes int64  `json:"sizeBytes"`
	Meta
}

func (h *Handler) initUpload(w http.ResponseWriter, r *http.Request) {
	actor, ok := h.actor(r)
	if !ok {
		unauthorized(w, r)
		return
	}
	var req initUploadRequest
	if err := httpjson.Decode(w, r, &req); err != nil {
		httpresponse.Fail(w, r, http.StatusBadRequest, httpresponse.ErrValidationFailed, "malformed JSON body")
		return
	}
	session, err := h.svc.InitUpload(r.Context(), actor.UserID, actor.RoleLevel, req.FileName, req.SizeBytes, req.Meta)
	if err != nil {
		writeError(w, r, err)
		return
	}
	httpresponse.OK(w, r, http.StatusCreated, map[string]any{"upload": session})
}

func (h *Handler) uploadStatus(w http.ResponseWriter, r *http.Request) {
	actor, ok := h.actor(r)
	if !ok {
		unauthorized(w, r)
		return
	}
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		notFound(w, r)
		return
	}
	session, err := h.svc.UploadStatus(r.Context(), actor.UserID, id)
	if err != nil {
		writeError(w, r, err)
		return
	}
	httpresponse.OK(w, r, http.StatusOK, map[string]any{"upload": session})
}

func (h *Handler) putChunk(w http.ResponseWriter, r *http.Request) {
	actor, ok := h.actor(r)
	if !ok {
		unauthorized(w, r)
		return
	}
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		notFound(w, r)
		return
	}
	idx, err := strconv.Atoi(r.URL.Query().Get("index"))
	if err != nil || idx < 0 {
		httpresponse.Fail(w, r, http.StatusBadRequest, httpresponse.ErrValidationFailed, "index must be a non-negative integer")
		return
	}
	_, chunkBytes := h.svc.Limits(actor.RoleLevel)
	data, err := io.ReadAll(http.MaxBytesReader(w, r.Body, int64(chunkBytes)+1024))
	if err != nil {
		httpresponse.Fail(w, r, http.StatusRequestEntityTooLarge, httpresponse.ErrValidationFailed, "chunk too large")
		return
	}
	if err := h.svc.PutChunk(r.Context(), actor.UserID, id, idx, data); err != nil {
		writeError(w, r, err)
		return
	}
	httpresponse.OK(w, r, http.StatusOK, map[string]any{"stored": true, "index": idx})
}

func (h *Handler) completeUpload(w http.ResponseWriter, r *http.Request) {
	actor, ok := h.actor(r)
	if !ok {
		unauthorized(w, r)
		return
	}
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		notFound(w, r)
		return
	}
	dto, err := h.svc.CompleteUpload(r.Context(), actor.UserID, id)
	if err != nil {
		writeError(w, r, err)
		return
	}
	httpresponse.OK(w, r, http.StatusCreated, map[string]any{"attachment": dto})
}

func (h *Handler) serve(w http.ResponseWriter, r *http.Request) {
	actor, ok := h.actor(r)
	if !ok {
		unauthorized(w, r)
		return
	}
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		notFound(w, r)
		return
	}
	b, err := h.svc.Fetch(r.Context(), id, actor.UserID, actor.RoleLevel)
	if err != nil {
		notFound(w, r)
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

// --- helpers ---

func unauthorized(w http.ResponseWriter, r *http.Request) {
	httpresponse.Fail(w, r, http.StatusUnauthorized, httpresponse.ErrAuthInvalidToken, "authentication required")
}

func notFound(w http.ResponseWriter, r *http.Request) {
	httpresponse.Fail(w, r, http.StatusNotFound, httpresponse.ErrResourceNotFound, "attachment not found")
}

func writeError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, ErrTooLarge):
		httpresponse.Fail(w, r, http.StatusRequestEntityTooLarge, httpresponse.ErrValidationFailed, "file too large")
	case errors.Is(err, ErrEmpty):
		httpresponse.Fail(w, r, http.StatusBadRequest, httpresponse.ErrValidationFailed, "empty file")
	case errors.Is(err, ErrBadMeta):
		httpresponse.Fail(w, r, http.StatusBadRequest, httpresponse.ErrValidationFailed, "invalid attachment metadata")
	case errors.Is(err, ErrBlockedType):
		httpresponse.Fail(w, r, http.StatusUnsupportedMediaType, httpresponse.ErrValidationFailed, "file type is not allowed")
	case errors.Is(err, ErrBadChunk):
		httpresponse.Fail(w, r, http.StatusBadRequest, httpresponse.ErrValidationFailed, "invalid chunk")
	case errors.Is(err, ErrIncomplete):
		httpresponse.Fail(w, r, http.StatusConflict, httpresponse.ErrValidationFailed, "upload is missing chunks")
	case errors.Is(err, ErrNotFound):
		notFound(w, r)
	default:
		httpresponse.Fail(w, r, http.StatusInternalServerError, httpresponse.ErrInternal, "upload failed")
	}
}
