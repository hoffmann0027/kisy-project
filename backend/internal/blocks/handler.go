package blocks

import (
	"context"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"kisy-backend/pkg/httpresponse"
)

// Handler exposes blocking to the app: block, unblock, and the list a person
// keeps. Mounted under /users (see cmd/server).
type Handler struct {
	svc   *Service
	actor func(*http.Request) (ActorMeta, bool)
	// profile turns an id into the public profile, so the list shows people
	// rather than identifiers. Nil: the list carries ids only.
	profile func(ctx context.Context, id uuid.UUID) (any, bool)
}

func NewHandler(svc *Service, actor func(*http.Request) (ActorMeta, bool)) *Handler {
	return &Handler{svc: svc, actor: actor}
}

// SetProfileLoader wires public-profile lookup for the blocked list.
func (h *Handler) SetProfileLoader(f func(ctx context.Context, id uuid.UUID) (any, bool)) {
	h.profile = f
}

func (h *Handler) Routes(r chi.Router) {
	r.Get("/me/blocks", h.list)
	r.Post("/{userID}/block", h.block)
	r.Delete("/{userID}/block", h.unblock)
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	actor, ok := h.actor(r)
	if !ok {
		unauthorized(w, r)
		return
	}
	list, err := h.svc.List(r.Context(), actor.UserID)
	if err != nil {
		httpresponse.Fail(w, r, http.StatusInternalServerError, httpresponse.ErrInternal, "internal error")
		return
	}
	out := make([]map[string]any, 0, len(list))
	for _, b := range list {
		row := map[string]any{"userId": b.UserID, "createdAt": b.CreatedAt}
		if h.profile != nil {
			if profile, ok := h.profile(r.Context(), b.UserID); ok {
				row["user"] = profile
			}
		}
		out = append(out, row)
	}
	httpresponse.OK(w, r, http.StatusOK, map[string]any{"blocks": out})
}

func (h *Handler) block(w http.ResponseWriter, r *http.Request)   { h.change(w, r, true) }
func (h *Handler) unblock(w http.ResponseWriter, r *http.Request) { h.change(w, r, false) }

func (h *Handler) change(w http.ResponseWriter, r *http.Request, block bool) {
	actor, ok := h.actor(r)
	if !ok {
		unauthorized(w, r)
		return
	}
	target, err := uuid.Parse(chi.URLParam(r, "userID"))
	if err != nil {
		httpresponse.Fail(w, r, http.StatusBadRequest, httpresponse.ErrValidationFailed, "userID must be a valid UUID")
		return
	}

	if block {
		err = h.svc.Add(r.Context(), actor, target)
	} else {
		err = h.svc.Remove(r.Context(), actor, target)
	}
	switch {
	case errors.Is(err, ErrSelf):
		httpresponse.Fail(w, r, http.StatusBadRequest, httpresponse.ErrValidationFailed, "нельзя заблокировать себя")
	case err != nil:
		httpresponse.Fail(w, r, http.StatusInternalServerError, httpresponse.ErrInternal, "internal error")
	default:
		httpresponse.OK(w, r, http.StatusOK, map[string]any{"blocked": block})
	}
}

func unauthorized(w http.ResponseWriter, r *http.Request) {
	httpresponse.Fail(w, r, http.StatusUnauthorized, httpresponse.ErrAuthInvalidToken, "authentication required")
}
