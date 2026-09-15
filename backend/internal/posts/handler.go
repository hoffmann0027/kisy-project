package posts

import (
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"kisy-backend/pkg/httpjson"
	"kisy-backend/pkg/httpresponse"
)

// defaultPageSize / maxPageSize bound a feed page.
const (
	defaultPageSize = 20
	maxPageSize     = 50
	// maxMediaUpload bounds one uploaded file at the transport level; the
	// blob store applies the real policy.
	maxMediaUpload = 25 << 20 // 25 MiB
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
	raw, err := io.ReadAll(http.MaxBytesReader(w, r.Body, maxMediaUpload))
	if err != nil {
		httpresponse.Fail(w, r, http.StatusRequestEntityTooLarge, httpresponse.ErrValidationFailed, "file too large")
		return
	}
	dto, err := h.svc.AttachMedia(r.Context(), postID, UploadedFile{
		FileName: r.Header.Get("X-File-Name"),
		MimeType: r.Header.Get("Content-Type"),
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
	raw, mime, err := h.svc.ReadMedia(r.Context(), mediaID, actor)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	w.Header().Set("Content-Type", mime)
	w.Header().Set("Cache-Control", "private, max-age=3600")
	w.Header().Set("X-Content-Type-Options", "nosniff")
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
