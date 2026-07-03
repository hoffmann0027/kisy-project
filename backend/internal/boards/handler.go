package boards

import (
	"errors"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"kisy-backend/pkg/httpjson"
	"kisy-backend/pkg/httpresponse"
)

// Handler exposes board endpoints. RequireAuth is applied by the router.
type Handler struct {
	svc   *Service
	actor func(*http.Request) (Actor, bool)
}

func NewHandler(svc *Service, actor func(*http.Request) (Actor, bool)) *Handler {
	return &Handler{svc: svc, actor: actor}
}

// GroupRoutes are mounted under /groups/{groupID}.
func (h *Handler) GroupRoutes(r chi.Router) {
	r.Get("/{groupID}/board", h.getBoard)
	r.Post("/{groupID}/board", h.createBoard)
}

// BoardRoutes are mounted under /boards.
func (h *Handler) BoardRoutes(r chi.Router) {
	r.Post("/{boardID}/columns", h.addColumn)
	r.Patch("/columns/{columnID}", h.renameColumn)
	r.Delete("/columns/{columnID}", h.deleteColumn)
	r.Post("/columns/{columnID}/cards", h.createCard)
	r.Patch("/cards/{cardID}", h.updateCard)
	r.Post("/cards/{cardID}/move", h.moveCard)
	r.Delete("/cards/{cardID}", h.deleteCard)
}

func (h *Handler) getBoard(w http.ResponseWriter, r *http.Request) {
	actor, ok := h.actor(r)
	if !ok {
		unauth(w, r)
		return
	}
	groupID, ok := parseID(w, r, "groupID")
	if !ok {
		return
	}
	board, err := h.svc.Get(r.Context(), groupID, actor)
	if errors.Is(err, ErrNotFound) {
		// No board yet: 404 with a marker the client uses to offer creation.
		httpresponse.Fail(w, r, http.StatusNotFound, httpresponse.ErrResourceNotFound, "board not found")
		return
	}
	if err != nil {
		writeErr(w, r, err)
		return
	}
	httpresponse.OK(w, r, http.StatusOK, map[string]any{"board": board})
}

type createBoardRequest struct {
	Title string `json:"title"`
}

func (h *Handler) createBoard(w http.ResponseWriter, r *http.Request) {
	actor, ok := h.actor(r)
	if !ok {
		unauth(w, r)
		return
	}
	groupID, ok := parseID(w, r, "groupID")
	if !ok {
		return
	}
	var req createBoardRequest
	_ = httpjson.Decode(w, r, &req)
	board, err := h.svc.Create(r.Context(), groupID, req.Title, actor)
	if err != nil {
		writeErr(w, r, err)
		return
	}
	httpresponse.OK(w, r, http.StatusCreated, map[string]any{"board": board})
}

type titleRequest struct {
	Title string `json:"title"`
}

func (h *Handler) addColumn(w http.ResponseWriter, r *http.Request) {
	actor, ok := h.actor(r)
	if !ok {
		unauth(w, r)
		return
	}
	boardID, ok := parseID(w, r, "boardID")
	if !ok {
		return
	}
	var req titleRequest
	if err := httpjson.Decode(w, r, &req); err != nil {
		badJSON(w, r)
		return
	}
	writeOK(w, r, h.svc.AddColumn(r.Context(), boardID, req.Title, actor))
}

func (h *Handler) renameColumn(w http.ResponseWriter, r *http.Request) {
	actor, ok := h.actor(r)
	if !ok {
		unauth(w, r)
		return
	}
	columnID, ok := parseID(w, r, "columnID")
	if !ok {
		return
	}
	var req titleRequest
	if err := httpjson.Decode(w, r, &req); err != nil {
		badJSON(w, r)
		return
	}
	writeOK(w, r, h.svc.RenameColumn(r.Context(), columnID, req.Title, actor))
}

func (h *Handler) deleteColumn(w http.ResponseWriter, r *http.Request) {
	actor, ok := h.actor(r)
	if !ok {
		unauth(w, r)
		return
	}
	columnID, ok := parseID(w, r, "columnID")
	if !ok {
		return
	}
	writeOK(w, r, h.svc.DeleteColumn(r.Context(), columnID, actor))
}

type cardRequest struct {
	Title       string  `json:"title"`
	Description *string `json:"description"`
	AssigneeID  *string `json:"assigneeId"`
	Label       *string `json:"label"`
	DueDate     *string `json:"dueDate"`
}

func (req cardRequest) toInput() (CardInput, error) {
	in := CardInput{Title: req.Title, Description: req.Description, Label: req.Label}
	if req.AssigneeID != nil && *req.AssigneeID != "" {
		id, err := uuid.Parse(*req.AssigneeID)
		if err != nil {
			return in, ErrInvalidInput
		}
		in.AssigneeID = &id
	}
	if req.DueDate != nil && *req.DueDate != "" {
		t, err := time.Parse(time.RFC3339, *req.DueDate)
		if err != nil {
			return in, ErrInvalidInput
		}
		in.DueDate = &t
	}
	return in, nil
}

func (h *Handler) createCard(w http.ResponseWriter, r *http.Request) {
	actor, ok := h.actor(r)
	if !ok {
		unauth(w, r)
		return
	}
	columnID, ok := parseID(w, r, "columnID")
	if !ok {
		return
	}
	var req cardRequest
	if err := httpjson.Decode(w, r, &req); err != nil {
		badJSON(w, r)
		return
	}
	in, err := req.toInput()
	if err != nil {
		writeErr(w, r, err)
		return
	}
	card, err := h.svc.CreateCard(r.Context(), columnID, in, actor)
	if err != nil {
		writeErr(w, r, err)
		return
	}
	httpresponse.OK(w, r, http.StatusCreated, map[string]any{"card": card})
}

func (h *Handler) updateCard(w http.ResponseWriter, r *http.Request) {
	actor, ok := h.actor(r)
	if !ok {
		unauth(w, r)
		return
	}
	cardID, ok := parseID(w, r, "cardID")
	if !ok {
		return
	}
	var req cardRequest
	if err := httpjson.Decode(w, r, &req); err != nil {
		badJSON(w, r)
		return
	}
	in, err := req.toInput()
	if err != nil {
		writeErr(w, r, err)
		return
	}
	writeOK(w, r, h.svc.UpdateCard(r.Context(), cardID, in, actor))
}

type moveRequest struct {
	ColumnID string `json:"columnId"`
	Index    int    `json:"index"`
}

func (h *Handler) moveCard(w http.ResponseWriter, r *http.Request) {
	actor, ok := h.actor(r)
	if !ok {
		unauth(w, r)
		return
	}
	cardID, ok := parseID(w, r, "cardID")
	if !ok {
		return
	}
	var req moveRequest
	if err := httpjson.Decode(w, r, &req); err != nil {
		badJSON(w, r)
		return
	}
	columnID, err := uuid.Parse(req.ColumnID)
	if err != nil {
		httpresponse.Fail(w, r, http.StatusBadRequest, httpresponse.ErrValidationFailed, "columnId must be a valid UUID")
		return
	}
	writeOK(w, r, h.svc.Move(r.Context(), cardID, columnID, req.Index, actor))
}

func (h *Handler) deleteCard(w http.ResponseWriter, r *http.Request) {
	actor, ok := h.actor(r)
	if !ok {
		unauth(w, r)
		return
	}
	cardID, ok := parseID(w, r, "cardID")
	if !ok {
		return
	}
	writeOK(w, r, h.svc.DeleteCard(r.Context(), cardID, actor))
}

// --- helpers ---

func parseID(w http.ResponseWriter, r *http.Request, name string) (uuid.UUID, bool) {
	id, err := uuid.Parse(chi.URLParam(r, name))
	if err != nil {
		httpresponse.Fail(w, r, http.StatusNotFound, httpresponse.ErrResourceNotFound, "not found")
		return uuid.Nil, false
	}
	return id, true
}

func writeOK(w http.ResponseWriter, r *http.Request, err error) {
	if err != nil {
		writeErr(w, r, err)
		return
	}
	httpresponse.OK(w, r, http.StatusOK, map[string]any{"ok": true})
}

func writeErr(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, ErrForbidden):
		httpresponse.Fail(w, r, http.StatusForbidden, httpresponse.ErrAccessDenied, "not permitted")
	case errors.Is(err, ErrNotFound), errors.Is(err, ErrColumnMissing):
		httpresponse.Fail(w, r, http.StatusNotFound, httpresponse.ErrResourceNotFound, "not found")
	case errors.Is(err, ErrBoardExists):
		httpresponse.Fail(w, r, http.StatusConflict, httpresponse.ErrValidationFailed, "board already exists")
	case errors.Is(err, ErrInvalidInput):
		httpresponse.Fail(w, r, http.StatusBadRequest, httpresponse.ErrValidationFailed, "invalid input")
	default:
		httpresponse.Fail(w, r, http.StatusInternalServerError, httpresponse.ErrInternal, "internal error")
	}
}

func unauth(w http.ResponseWriter, r *http.Request) {
	httpresponse.Fail(w, r, http.StatusUnauthorized, httpresponse.ErrAuthInvalidToken, "authentication required")
}

func badJSON(w http.ResponseWriter, r *http.Request) {
	httpresponse.Fail(w, r, http.StatusBadRequest, httpresponse.ErrValidationFailed, "malformed JSON body")
}
