package admin_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"kisy-backend/internal/admin"
	"kisy-backend/internal/auth"
	"kisy-backend/internal/auth/token"
	"kisy-backend/internal/moderation"
)

// Every route under /admin is the CEO's alone — including every sanction
// endpoint (warn, mute, delete, revoke, restore, the lists). Rather than listing the routes
// here — a list that silently stops covering the next endpoint someone adds —
// the test walks the router admin.Mount actually builds, so a new route is
// checked the moment it exists.

func routerAs(claims *token.AccessClaims) http.Handler {
	r := chi.NewRouter()
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			next.ServeHTTP(w, req.WithContext(auth.ContextWithClaims(req.Context(), claims)))
		})
	})
	// No service behind the handler: a request that got past the gates would
	// panic on it, which fails the test just as loudly as a 200 would.
	moderationRoutes := moderation.NewHandler(nil, func(*http.Request) (moderation.ActorMeta, bool) {
		return moderation.ActorMeta{UserID: claims.UserID, RoleLevel: claims.RoleLevel}, true
	}).AdminRoutes
	admin.Mount(r, auth.NewMiddleware(nil, nil, nil), admin.NewHandler(nil, nil, func(*http.Request) (admin.ActorMeta, bool) {
		return admin.ActorMeta{UserID: claims.UserID}, true
	}), moderationRoutes)
	return r
}

type route struct{ method, path string }

func adminRoutes(t *testing.T) []route {
	t.Helper()
	var out []route
	walker, ok := routerAs(&token.AccessClaims{}).(chi.Routes)
	if !ok {
		t.Fatal("router is not walkable")
	}
	err := chi.Walk(walker, func(method, path string, _ http.Handler, _ ...func(http.Handler) http.Handler) error {
		if strings.HasPrefix(path, "/admin") {
			// Fill path parameters with something that parses, so a handler
			// reached by mistake would get far enough to show it.
			path = strings.NewReplacer("{userID}", uuid.NewString(), "{groupID}", uuid.NewString(), "{sanctionID}", uuid.NewString()).Replace(path)
			out = append(out, route{method, path})
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	// Guard against passing because nothing was walked.
	// 9 admin routes + 6 moderation routes (warn/mute/delete, revoke, restore,
	// lists): the sanction endpoints are part of what this proves.
	if len(out) < 15 {
		t.Fatalf("walked only %d admin routes", len(out))
	}
	return out
}

func TestEveryAdminRouteRefusesAnyoneButTheCEO(t *testing.T) {
	outsiders := map[string]*token.AccessClaims{
		"level 2 (the strongest non-CEO)": {UserID: uuid.New(), RoleLevel: 2, Kind: "invited"},
		"level 10":                        {UserID: uuid.New(), RoleLevel: 10, Kind: "invited"},
		"basic account (no level)":        {UserID: uuid.New(), RoleLevel: 0, Kind: "basic"},
	}
	for name, claims := range outsiders {
		router := routerAs(claims)
		for _, rt := range adminRoutes(t) {
			rec := httptest.NewRecorder()
			req := httptest.NewRequest(rt.method, rt.path, strings.NewReader(`{}`))
			func() {
				defer func() {
					if p := recover(); p != nil {
						t.Errorf("%s: %s %s reached the handler (%v)", name, rt.method, rt.path, p)
					}
				}()
				router.ServeHTTP(rec, req)
			}()
			if rec.Code != http.StatusForbidden {
				t.Errorf("%s: %s %s = %d, want 403", name, rt.method, rt.path, rec.Code)
			}
		}
	}
}
