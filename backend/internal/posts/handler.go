package posts

import (
	"errors"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"kisy-backend/internal/quota"
	"kisy-backend/pkg/httpjson"
	"kisy-backend/pkg/httpresponse"
)

// defaultPageSize / maxPageSize bound a feed page.
const (
	defaultPageSize = 20
	maxPageSize     = 50
)

type Handler struct {
	svc   *Service
	actor func(*http.Request) (ActorMeta, bool)
}

func NewHandler(svc *Service, actor func(*http.Request) (ActorMeta, bool)) *Handler {
	return &Handler{svc: svc, actor: actor}
}

// Routes registers the post endpoints that hang off /posts. The
// community-scoped ones are registered by the groups router.
func (h *Handler) Routes(r chi.Router) {
	r.Delete("/{postID}", h.delete)
	r.Post("/{postID}/media", h.uploadMedia)
	r.Post("/{postID}/reactions", h.react)
	r.Delete("/{postID}/reactions", h.unreact)
	r.Get("/media/{mediaID}", h.serveMedia)
}

// FeedRoutes registers /feed.
func (h *Handler) FeedRoutes(r chi.Router) {
	r.Get("/", h.feed)
	r.Post("/hidden/{communityID}", h.hide)
	r.Delete("/hidden/{communityID}", h.show)
}

// CommunityRoutes registers the endpoints scoped to one community.
func (h *Handler) CommunityRoutes(r chi.Router) {
	r.Get("/", h.listCommunity)
	r.Post("/", h.create)
}

type createRequest struct {
	Text string `json:"text"`
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	actor, ok := h.actor(r)
	if !ok {
		unauthorized(w, r)
		return
	}
	communityID, err := uuid.Parse(chi.URLParam(r, "groupID"))
	if err != nil {
		notFound(w, r)
		return
	}
	var req createRequest
	if err := httpjson.Decode(w, r, &req); err != nil {
		httpresponse.Fail(w, r, http.StatusBadRequest, httpresponse.ErrValidationFailed, "malformed JSON body")
		return
	}

	// Media is uploaded to the post afterwards, not with it: a file and a JSON
	// body do not travel together, and a draft that could hold files before a
	// post exists would be state to expire and clean up. The composer keeps
	// the post to itself until its uploads finish.
	dto, err := h.svc.Create(r.Context(), CreateInput{CommunityID: communityID, Text: req.Text}, actor)
	h.write(w, r, dto, err, http.StatusCreated)
}

func (h *Handler) listCommunity(w http.ResponseWriter, r *http.Request) {
	actor, ok := h.actor(r)
	if !ok {
		unauthorized(w, r)
		return
	}
	communityID, err := uuid.Parse(chi.URLParam(r, "groupID"))
	if err != nil {
		notFound(w, r)
		return
	}
	page, err := h.svc.ListCommunity(r.Context(), communityID, r.URL.Query().Get("cursor"), pageSize(r), actor)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httpresponse.OK(w, r, http.StatusOK, page)
}

func (h *Handler) feed(w http.ResponseWriter, r *http.Request) {
	actor, ok := h.actor(r)
	if !ok {
		unauthorized(w, r)
		return
	}
	sort := r.URL.Query().Get("sort")
	if sort != "new" {
		sort = "popular"
	}
	page, err := h.svc.Feed(r.Context(), sort, r.URL.Query().Get("cursor"), pageSize(r), actor)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httpresponse.OK(w, r, http.StatusOK, page)
}

func (h *Handler) delete(w http.ResponseWriter, r *http.Request) {
	actor, ok := h.actor(r)
	if !ok {
		unauthorized(w, r)
		return
	}
	postID, err := uuid.Parse(chi.URLParam(r, "postID"))
	if err != nil {
		notFound(w, r)
		return
	}
	if err := h.svc.Delete(r.Context(), postID, actor); err != nil {
		h.fail(w, r, err)
		return
	}
	httpresponse.OK(w, r, http.StatusOK, map[string]any{"deleted": true})
}

type reactionRequest struct {
	Emoji string `json:"emoji"`
}

func (h *Handler) react(w http.ResponseWriter, r *http.Request)   { h.reaction(w, r, true) }
func (h *Handler) unreact(w http.ResponseWriter, r *http.Request) { h.reaction(w, r, false) }

func (h *Handler) reaction(w http.ResponseWriter, r *http.Request, on bool) {
	actor, ok := h.actor(r)
	if !ok {
		unauthorized(w, r)
		return
	}
	postID, err := uuid.Parse(chi.URLParam(r, "postID"))
	if err != nil {
		notFound(w, r)
		return
	}
	var req reactionRequest
	if err := httpjson.Decode(w, r, &req); err != nil {
		httpresponse.Fail(w, r, http.StatusBadRequest, httpresponse.ErrValidationFailed, "malformed JSON body")
		return
	}
	if err := h.svc.React(r.Context(), postID, strings.TrimSpace(req.Emoji), on, actor); err != nil {
		h.fail(w, r, err)
		return
	}
	httpresponse.OK(w, r, http.StatusOK, map[string]any{"ok": true})
}

func (h *Handler) hide(w http.ResponseWriter, r *http.Request) { h.hidden(w, r, true) }
func (h *Handler) show(w http.ResponseWriter, r *http.Request) { h.hidden(w, r, false) }

func (h *Handler) hidden(w http.ResponseWriter, r *http.Request, hide bool) {
	actor, ok := h.actor(r)
	if !ok {
		unauthorized(w, r)
		return
	}
	communityID, err := uuid.Parse(chi.URLParam(r, "communityID"))
	if err != nil {
		notFound(w, r)
		return
	}
	if err := h.svc.HideCommunity(r.Context(), communityID, hide, actor); err != nil {
		h.fail(w, r, err)
		return
	}
	httpresponse.OK(w, r, http.StatusOK, map[string]any{"hidden": hide})
}

func (h *Handler) uploadMedia(w http.ResponseWriter, r *http.Request) {
	actor, ok := h.actor(r)
	if !ok {
		unauthorized(w, r)
		return
	}
	postID, err := uuid.Parse(chi.URLParam(r, "postID"))
	if err != nil {
		notFound(w, r)
		return
	}
	raw, err := io.ReadAll(http.MaxBytesReader(w, r.Body, h.svc.MaxMediaBytes()))
	if err != nil {
		httpresponse.Fail(w, r, http.StatusRequestEntityTooLarge, httpresponse.ErrValidationFailed, "file too large")
		return
	}
	dto, err := h.svc.AttachMedia(r.Context(), postID, UploadedFile{
		FileName: r.Header.Get("X-File-Name"),
		Bytes:    raw,
	}, actor)
	h.write(w, r, dto, err, http.StatusOK)
}

func (h *Handler) serveMedia(w http.ResponseWriter, r *http.Request) {
	actor, ok := h.actor(r)
	if !ok {
		unauthorized(w, r)
		return
	}
	mediaID, err := uuid.Parse(chi.URLParam(r, "mediaID"))
	if err != nil {
		notFound(w, r)
		return
	}
	raw, mime, name, err := h.svc.ReadMedia(r.Context(), mediaID, actor)
	if err != nil {
		h.fail(w, r, err)
		return
	}

	// The type was sniffed from the bytes at upload, never taken from the
	// uploader's header, so it cannot be used to smuggle active content.
	//
	// Pictures, audio and video render in place. Everything else is served as
	// an opaque download with a neutral type — not as whatever it claims to
	// be. An uploaded .html handed back as text/html from this origin would be
	// stored XSS on the one screen every member of a community looks at, and
	// the file the browser never interprets cannot be one.
	disposition := "attachment"
	serveType := "application/octet-stream"
	if strings.HasPrefix(mime, "image/") || strings.HasPrefix(mime, "audio/") || strings.HasPrefix(mime, "video/") {
		disposition = "inline"
		serveType = mime
	}
	w.Header().Set("Content-Type", serveType)
	w.Header().Set("Content-Length", strconv.Itoa(len(raw)))
	w.Header().Set("Cache-Control", "private, max-age=3600")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Content-Disposition", disposition+`; filename*=UTF-8''`+url.PathEscape(name))
	// #nosec G705 -- these are user-uploaded bytes, and the analysis is right
	// about that; what it cannot see is that they are never handed to the
	// browser as active content. The type is sniffed from the bytes rather
	// than believed from a header, anything that is not a picture, audio or
	// video is served as application/octet-stream with Content-Disposition:
	// attachment, and nosniff is set. Message attachments serve files the same
	// way. The filename defences have their own tests in media_test.go.
	_, _ = w.Write(raw)
}

func (h *Handler) write(w http.ResponseWriter, r *http.Request, dto *DTO, err error, status int) {
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httpresponse.OK(w, r, status, map[string]any{"post": dto})
}

func (h *Handler) fail(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, ErrNotFound):
		notFound(w, r)
	case errors.Is(err, ErrTooLarge):
		httpresponse.Fail(w, r, http.StatusRequestEntityTooLarge, httpresponse.ErrValidationFailed, "файл слишком большой")
	case quota.Is(err):
		status, msg, _ := quota.Describe(err)
		httpresponse.Fail(w, r, status, httpresponse.ErrQuotaExceeded, msg)
	case errors.Is(err, ErrMembersOnly):
		httpresponse.Fail(w, r, http.StatusForbidden, httpresponse.ErrAccessDenied, "сообщество закрытое: записи видят только участники")
	case errors.Is(err, ErrForbidden):
		httpresponse.Fail(w, r, http.StatusForbidden, httpresponse.ErrAccessDenied, "у вас нет прав публиковать здесь")
	case errors.Is(err, ErrNotCommunity):
		httpresponse.Fail(w, r, http.StatusBadRequest, httpresponse.ErrValidationFailed, "это группа, а не сообщество")
	case errors.Is(err, ErrEmpty):
		httpresponse.Fail(w, r, http.StatusBadRequest, httpresponse.ErrValidationFailed, "пост не может быть пустым")
	case errors.Is(err, ErrTooLong):
		httpresponse.Fail(w, r, http.StatusBadRequest, httpresponse.ErrValidationFailed, "пост слишком длинный")
	default:
		httpresponse.Fail(w, r, http.StatusInternalServerError, httpresponse.ErrInternal, "internal error")
	}
}

func unauthorized(w http.ResponseWriter, r *http.Request) {
	httpresponse.Fail(w, r, http.StatusUnauthorized, httpresponse.ErrAuthInvalidToken, "authentication required")
}

// A post in a community the caller cannot see is masked as missing, never as
// forbidden: "you may not see this" still tells them it exists.
func notFound(w http.ResponseWriter, r *http.Request) {
	httpresponse.Fail(w, r, http.StatusNotFound, httpresponse.ErrResourceNotFound, "post not found")
}

func pageSize(r *http.Request) int {
	n, err := strconv.Atoi(r.URL.Query().Get("limit"))
	if err != nil || n <= 0 {
		return defaultPageSize
	}
	if n > maxPageSize {
		return maxPageSize
	}
	return n
}
