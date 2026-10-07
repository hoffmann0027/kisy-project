//go:build integration

package main

import (
	"bytes"
	"context"
	"image"
	"image/png"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"kisy-backend/internal/audit"
	"kisy-backend/internal/auth"
	"kisy-backend/internal/auth/token"
	"kisy-backend/internal/avatars"
	"kisy-backend/internal/groups"
	"kisy-backend/internal/platform/testdb"
)

// Audit A-36: a group's avatar was served to anyone signed in, whatever the
// group — a basic or lower-level account got the picture of a group above its
// clearance, and 200-or-404 told it which group ids exist.
func TestAGroupAvatarFollowsTheGroupsVisibility(t *testing.T) {
	pool := testdb.New(t)
	ctx := context.Background()
	groupsSvc := groups.NewService(pool, groups.NewPostgresRepository(), audit.NewPostgresRecorder(quiet()))
	avatarsSvc := avatars.NewService(pool, avatars.NewPostgresRepository())

	lead := testdb.SeedUser(t, pool, "lead_"+uuid.NewString()[:6], 3)
	g, err := groupsSvc.Create(ctx, groups.CreateInput{Name: "Руководство", MinRoleLevel: 3}, groups.ActorMeta{UserID: lead, RoleLevel: 3})
	if err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, image.NewRGBA(image.Rect(0, 0, 128, 128))); err != nil {
		t.Fatal(err)
	}
	if _, err := avatarsSvc.Store(ctx, avatars.OwnerGroup, g.ID, buf.Bytes()); err != nil {
		t.Fatal(err)
	}

	h := avatars.NewHandler(avatarsSvc)
	// As cmd/server wires it.
	h.SetGroupGuard(func(r *http.Request, groupID uuid.UUID) bool {
		claims, ok := auth.ClaimsFromContext(r.Context())
		if !ok {
			return false
		}
		_, err := groupsSvc.Get(r.Context(), groupID, groups.ActorMeta{UserID: claims.UserID, RoleLevel: claims.RoleLevel})
		return err == nil
	})
	router := chi.NewRouter()
	h.Routes(router)

	fetch := func(level int) int {
		req := httptest.NewRequest(http.MethodGet, "/avatars/group/"+g.ID.String(), nil)
		req = req.WithContext(auth.ContextWithClaims(req.Context(), &token.AccessClaims{UserID: uuid.New(), RoleLevel: level}))
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		return rec.Code
	}

	if got := fetch(5); got != http.StatusNotFound {
		t.Fatalf("a level-5 account got the avatar of a level-3 group: %d", got)
	}
	if got := fetch(0); got != http.StatusNotFound {
		t.Fatalf("a basic account got the avatar of a level-3 group: %d", got)
	}
	if got := fetch(3); got != http.StatusOK {
		t.Fatalf("someone cleared for the group could not see its avatar: %d", got)
	}
}
