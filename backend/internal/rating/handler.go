package rating

import (
	"encoding/csv"
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"kisy-backend/pkg/httpjson"
	"kisy-backend/pkg/httpresponse"
)

// Handler exposes /rating. RequireAuth is applied by the router.
type Handler struct {
	svc   *Service
	actor func(*http.Request) (Actor, bool)
}

func NewHandler(svc *Service, actor func(*http.Request) (Actor, bool)) *Handler {
	return &Handler{svc: svc, actor: actor}
}

func (h *Handler) Routes(r chi.Router) {
	r.Get("/board", h.board)
	r.Get("/analytics", h.analytics)
	r.Get("/export.csv", h.exportCSV)
	r.Post("/projects", h.createProject)
	r.Patch("/projects/{id}/level", h.setProjectLevel)
	r.Delete("/projects/{id}", h.deleteProject)
	r.Post("/projects/{id}/tasks", h.createTask)
	r.Post("/projects/{id}/finance", h.addFinance)
	r.Post("/tasks/{id}/assign", h.assign)
	r.Patch("/tasks/{id}/progress", h.setProgress)
	r.Post("/tasks/{id}/return", h.returnTask)
	r.Delete("/tasks/{id}", h.deleteTask)
}

func (h *Handler) returnTask(w http.ResponseWriter, r *http.Request) {
	actor, ok := h.auth(w, r)
	if !ok {
		return
	}
	id, err := taskID(r)
	if err != nil {
		httpresponse.Fail(w, r, http.StatusNotFound, httpresponse.ErrResourceNotFound, "task not found")
		return
	}
	if err := h.svc.ReturnTask(r.Context(), id, actor); err != nil {
		h.fail(w, r, err)
		return
	}
	httpresponse.OK(w, r, http.StatusOK, map[string]any{"returned": true})
}

func (h *Handler) deleteTask(w http.ResponseWriter, r *http.Request) {
	actor, ok := h.auth(w, r)
	if !ok {
		return
	}
	id, err := taskID(r)
	if err != nil {
		httpresponse.Fail(w, r, http.StatusNotFound, httpresponse.ErrResourceNotFound, "task not found")
		return
	}
	if err := h.svc.DeleteTask(r.Context(), id, actor); err != nil {
		h.fail(w, r, err)
		return
	}
	httpresponse.OK(w, r, http.StatusOK, map[string]any{"deleted": true})
}

// fail maps domain errors to HTTP responses.
func (h *Handler) fail(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, ErrForbidden):
		httpresponse.Fail(w, r, http.StatusForbidden, httpresponse.ErrAccessDenied, "action not permitted")
	case errors.Is(err, ErrNotFound):
		httpresponse.Fail(w, r, http.StatusNotFound, httpresponse.ErrResourceNotFound, "not found")
	case errors.Is(err, ErrValidation):
		httpresponse.Fail(w, r, http.StatusBadRequest, httpresponse.ErrValidationFailed, "invalid input")
	case errors.Is(err, ErrAlreadyClaimed):
		httpresponse.Fail(w, r, http.StatusConflict, httpresponse.ErrValidationFailed, "task already has an assignee")
	default:
		httpresponse.Fail(w, r, http.StatusInternalServerError, httpresponse.ErrInternal, "internal error")
	}
}

func (h *Handler) auth(w http.ResponseWriter, r *http.Request) (Actor, bool) {
	actor, ok := h.actor(r)
	if !ok {
		httpresponse.Fail(w, r, http.StatusUnauthorized, httpresponse.ErrAuthInvalidToken, "authentication required")
	}
	return actor, ok
}

func taskID(r *http.Request) (uuid.UUID, error) { return uuid.Parse(chi.URLParam(r, "id")) }

func (h *Handler) board(w http.ResponseWriter, r *http.Request) {
	actor, ok := h.auth(w, r)
	if !ok {
		return
	}
	board, err := h.svc.Board(r.Context(), actor.RoleLevel)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httpresponse.OK(w, r, http.StatusOK, board)
}

func (h *Handler) analytics(w http.ResponseWriter, r *http.Request) {
	actor, ok := h.auth(w, r)
	if !ok {
		return
	}
	a, err := h.svc.Analytics(r.Context(), actor.RoleLevel)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httpresponse.OK(w, r, http.StatusOK, a)
}

func rubles(kopecks int64) string {
	return strconv.FormatFloat(float64(kopecks)/100, 'f', 2, 64)
}

// exportCSV streams the profit ledger as a CSV download.
func (h *Handler) exportCSV(w http.ResponseWriter, r *http.Request) {
	actor, ok := h.auth(w, r)
	if !ok {
		return
	}
	rows, err := h.svc.ExportFinance(r.Context(), actor.RoleLevel)
	if err != nil {
		h.fail(w, r, err)
		return
	}

	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="kisy-profit.csv"`)
	// UTF-8 BOM so Excel opens Cyrillic correctly.
	_, _ = w.Write([]byte{0xEF, 0xBB, 0xBF})

	cw := csv.NewWriter(w)
	_ = cw.Write([]string{"Дата", "Проект", "Задача", "Доход", "Расход", "Прибыль", "Автор", "Комментарий"})
	for _, row := range rows {
		task, note := "", ""
		if row.TaskTitle != nil {
			task = *row.TaskTitle
		}
		if row.Note != nil {
			note = *row.Note
		}
		_ = cw.Write([]string{
			row.CreatedAt.Format("2006-01-02 15:04"),
			row.ProjectTitle,
			task,
			rubles(row.IncomeKopecks),
			rubles(row.ExpenseKopecks),
			rubles(row.IncomeKopecks - row.ExpenseKopecks),
			row.AuthorName,
			note,
		})
	}
	cw.Flush()
	if err := cw.Error(); err != nil {
		// Header already written; best we can do is log via a trailer-less error.
		_, _ = fmt.Fprint(w, "\n# error writing csv")
	}
}

type createProjectRequest struct {
	Title       string  `json:"title"`
	Description *string `json:"description"`
	Difficulty  string  `json:"difficulty"`
	MinLevel    int     `json:"minLevel"`
}

func (h *Handler) createProject(w http.ResponseWriter, r *http.Request) {
	actor, ok := h.auth(w, r)
	if !ok {
		return
	}
	var req createProjectRequest
	if err := httpjson.Decode(w, r, &req); err != nil {
		httpresponse.Fail(w, r, http.StatusBadRequest, httpresponse.ErrValidationFailed, "malformed JSON body")
		return
	}
	id, err := h.svc.CreateProject(r.Context(), CreateProjectInput{Title: req.Title, Description: req.Description, Difficulty: req.Difficulty, MinLevel: req.MinLevel}, actor)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httpresponse.OK(w, r, http.StatusCreated, map[string]any{"id": id})
}

type setLevelRequest struct {
	MinLevel int `json:"minLevel"`
}

func (h *Handler) setProjectLevel(w http.ResponseWriter, r *http.Request) {
	actor, ok := h.auth(w, r)
	if !ok {
		return
	}
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httpresponse.Fail(w, r, http.StatusNotFound, httpresponse.ErrResourceNotFound, "project not found")
		return
	}
	var req setLevelRequest
	if err := httpjson.Decode(w, r, &req); err != nil {
		httpresponse.Fail(w, r, http.StatusBadRequest, httpresponse.ErrValidationFailed, "malformed JSON body")
		return
	}
	if err := h.svc.SetProjectLevel(r.Context(), id, req.MinLevel, actor); err != nil {
		h.fail(w, r, err)
		return
	}
	httpresponse.OK(w, r, http.StatusOK, map[string]any{"ok": true})
}

func (h *Handler) deleteProject(w http.ResponseWriter, r *http.Request) {
	actor, ok := h.auth(w, r)
	if !ok {
		return
	}
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httpresponse.Fail(w, r, http.StatusNotFound, httpresponse.ErrResourceNotFound, "project not found")
		return
	}
	if err := h.svc.DeleteProject(r.Context(), id, actor); err != nil {
		h.fail(w, r, err)
		return
	}
	httpresponse.OK(w, r, http.StatusOK, map[string]any{"deleted": true})
}

type createTaskRequest struct {
	Title string `json:"title"`
}

func (h *Handler) createTask(w http.ResponseWriter, r *http.Request) {
	actor, ok := h.auth(w, r)
	if !ok {
		return
	}
	projectID, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httpresponse.Fail(w, r, http.StatusNotFound, httpresponse.ErrResourceNotFound, "project not found")
		return
	}
	var req createTaskRequest
	if err := httpjson.Decode(w, r, &req); err != nil {
		httpresponse.Fail(w, r, http.StatusBadRequest, httpresponse.ErrValidationFailed, "malformed JSON body")
		return
	}
	id, err := h.svc.CreateTask(r.Context(), projectID, req.Title, actor)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httpresponse.OK(w, r, http.StatusCreated, map[string]any{"id": id})
}

func (h *Handler) assign(w http.ResponseWriter, r *http.Request) {
	actor, ok := h.auth(w, r)
	if !ok {
		return
	}
	id, err := taskID(r)
	if err != nil {
		httpresponse.Fail(w, r, http.StatusNotFound, httpresponse.ErrResourceNotFound, "task not found")
		return
	}
	if err := h.svc.AssignSelf(r.Context(), id, actor); err != nil {
		h.fail(w, r, err)
		return
	}
	httpresponse.OK(w, r, http.StatusOK, map[string]any{"assigned": true})
}

type progressRequest struct {
	Progress int `json:"progress"`
}

func (h *Handler) setProgress(w http.ResponseWriter, r *http.Request) {
	actor, ok := h.auth(w, r)
	if !ok {
		return
	}
	id, err := taskID(r)
	if err != nil {
		httpresponse.Fail(w, r, http.StatusNotFound, httpresponse.ErrResourceNotFound, "task not found")
		return
	}
	var req progressRequest
	if err := httpjson.Decode(w, r, &req); err != nil {
		httpresponse.Fail(w, r, http.StatusBadRequest, httpresponse.ErrValidationFailed, "malformed JSON body")
		return
	}
	if err := h.svc.SetProgress(r.Context(), id, req.Progress, actor); err != nil {
		h.fail(w, r, err)
		return
	}
	httpresponse.OK(w, r, http.StatusOK, map[string]any{"ok": true})
}

type financeRequest struct {
	IncomeKopecks  int64   `json:"incomeKopecks"`
	ExpenseKopecks int64   `json:"expenseKopecks"`
	Note           *string `json:"note"`
}

func (h *Handler) addFinance(w http.ResponseWriter, r *http.Request) {
	actor, ok := h.auth(w, r)
	if !ok {
		return
	}
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httpresponse.Fail(w, r, http.StatusNotFound, httpresponse.ErrResourceNotFound, "project not found")
		return
	}
	var req financeRequest
	if err := httpjson.Decode(w, r, &req); err != nil {
		httpresponse.Fail(w, r, http.StatusBadRequest, httpresponse.ErrValidationFailed, "malformed JSON body")
		return
	}
	if err := h.svc.AddFinance(r.Context(), id, FinanceInput{IncomeKopecks: req.IncomeKopecks, ExpenseKopecks: req.ExpenseKopecks, Note: req.Note}, actor); err != nil {
		h.fail(w, r, err)
		return
	}
	httpresponse.OK(w, r, http.StatusCreated, map[string]any{"ok": true})
}
