//go:build integration

package main

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	yaml "go.yaml.in/yaml/v3"

	"kisy-backend/internal/config"
	"kisy-backend/internal/platform/testdb"
)

// Audit E-04: there was no HTTP-level authorization test outside /admin, and
// that is precisely where the HIGH findings of the September audit lived — a
// wall readable without membership, an avatar writable by anyone, search and
// WebSocket ignoring group visibility. Every one of them was a route whose
// gate was assumed rather than checked.
//
// This walks the real router, the one newRouter builds from the real modules,
// and demands a session for everything that is not deliberately public. A new
// endpoint mounted outside the authenticated group fails here the moment it
// exists, instead of in a report months later.

// publicRoutes are the endpoints that must work without a session, each with
// the reason. Anything not listed here is expected to answer 401.
var publicRoutes = map[string]string{
	"GET /health":                   "liveness for the platform's health check",
	"GET /ready":                    "readiness: pings Postgres and Redis",
	"GET /metrics":                  "closed in production by a token instead (audit A-18); see metricsHandler",
	"POST /api/v1/auth/register":    "sign-up",
	"POST /api/v1/auth/login":       "sign-in",
	"POST /api/v1/auth/refresh":     "token rotation: the access token is expected to be dead here",
	"GET /api/v1/auth/registration": "whether this deployment accepts sign-ups, asked before signing up",
	"GET /ws":                       "the handshake authenticates inside the upgrade (a query token)",
}

func testConfig(t *testing.T) *config.Config {
	t.Helper()
	for k, v := range map[string]string{
		"APP_ENV":            "development",
		"POSTGRES_PASSWORD":  "test-db-password",
		"REDIS_PASSWORD":     "test-redis-password",
		"JWT_ACCESS_SECRET":  strings.Repeat("a", 32),
		"JWT_REFRESH_SECRET": strings.Repeat("b", 32),
		"DATABASE_URL":       "",
		"REDIS_URL":          "",
		"TURNSTILE_ENABLED":  "false",
	} {
		t.Setenv(k, v)
	}
	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("config: %v", err)
	}
	return cfg
}

// fullRouter builds the production router over a throwaway database and the
// test Redis. Nothing is stubbed: the point is to exercise the wiring.
func fullRouter(t *testing.T) http.Handler {
	t.Helper()
	pool := testdb.New(t)
	rdb := testRedis(t)
	cfg := testConfig(t)

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	mods, err := buildModules(ctx, cfg, pool, rdb, log)
	if err != nil {
		t.Fatalf("buildModules: %v", err)
	}

	return newRouter(routerDeps{
		log:           log,
		pg:            pool,
		rdb:           rdb,
		mods:          mods,
		allowedOrigin: "http://localhost:5173",
		features:      cfg.Features(),
	})
}

// walkRoutes lists every method+path the router serves, with path parameters
// filled in so a handler reached by mistake gets far enough to show it.
func walkRoutes(t *testing.T, h http.Handler) []route {
	t.Helper()
	return walkRoutesRaw(t, h)
}

// walkRoutesRaw is the walk itself: route.path has its parameters replaced by
// real uuids (so a request can be made), route.raw keeps the pattern.
func walkRoutesRaw(t *testing.T, h http.Handler) []route {
	t.Helper()
	walker, ok := h.(chi.Routes)
	if !ok {
		t.Fatal("router is not walkable")
	}

	var out []route
	err := chi.Walk(walker, func(method, path string, _ http.Handler, _ ...func(http.Handler) http.Handler) error {
		// chi prints a wildcard subtree (the SPA fallback) as /*; there is no
		// request to make of it here.
		if strings.HasSuffix(path, "/*") {
			return nil
		}
		raw := path
		for strings.Contains(path, "{") {
			open := strings.Index(path, "{")
			close := strings.Index(path[open:], "}") + open
			if close <= open {
				break
			}
			path = path[:open] + uuid.NewString() + path[close+1:]
		}
		out = append(out, route{method: method, path: path, raw: raw})
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	// Guard against passing because nothing was walked.
	if len(out) < 100 {
		t.Fatalf("walked only %d routes: the router is not the real one", len(out))
	}
	return out
}

type route struct{ method, path, raw string }

// key is the form publicRoutes is written in: parameters back as they were.
func (r route) key() string {
	parts := strings.Split(r.path, "/")
	for i, p := range parts {
		if _, err := uuid.Parse(p); err == nil {
			parts[i] = "{id}"
		}
	}
	return r.method + " " + strings.Join(parts, "/")
}

func TestEveryRouteRequiresASessionUnlessItIsDeliberatelyPublic(t *testing.T) {
	router := fullRouter(t)
	routes := walkRoutes(t, router)

	checked := 0
	for _, rt := range routes {
		key := rt.key()
		if _, public := publicRoutes[key]; public {
			continue
		}
		checked++

		rec := httptest.NewRecorder()
		req := httptest.NewRequest(rt.method, rt.path, strings.NewReader(`{}`))
		req.Header.Set("Content-Type", "application/json")
		// Same-origin: the CSRF gate must not be what refuses the request, or
		// this test would pass while the route itself is wide open.
		req.Header.Set("Origin", "http://localhost:5173")

		func() {
			defer func() {
				if p := recover(); p != nil {
					t.Errorf("%s %s reached its handler without a session (%v)", rt.method, rt.path, p)
				}
			}()
			router.ServeHTTP(rec, req)
		}()

		if rec.Code != http.StatusUnauthorized {
			t.Errorf("%s %s = %d without a session, want 401 (add it to publicRoutes with a reason if that is intended)",
				rt.method, rt.path, rec.Code)
		}
	}

	if checked < 100 {
		t.Fatalf("only %d routes were checked", checked)
	}
	t.Logf("%d routes require a session, %d deliberately public", checked, len(routes)-checked)
}

// TestPublicRoutesAreStillThere is the other direction: the allowlist must not
// rot into a list of routes that no longer exist, because then it silences
// nothing and hides the next mistake.
func TestPublicRoutesAreStillThere(t *testing.T) {
	routes := walkRoutes(t, fullRouter(t))
	live := make(map[string]bool, len(routes))
	for _, rt := range routes {
		live[rt.key()] = true
	}
	for key := range publicRoutes {
		if !live[key] {
			t.Errorf("publicRoutes lists %q, which the router does not serve", key)
		}
	}
}

// Audit C-07 and the Definition of Done ("every endpoint documented in
// OpenAPI"): two call endpoints were missing from the spec, and nothing would
// have noticed the third. The comparison ignores parameter NAMES ({callID} vs
// {id}) and only insists that the endpoint is described at all.
func TestEveryRouteIsDocumentedInOpenAPI(t *testing.T) {
	documented := openapiOperations(t)
	routes := walkRoutesRaw(t, fullRouter(t))

	// Paths that are deliberately outside the documented API surface.
	skip := map[string]bool{
		"/health": true, "/ready": true, "/metrics": true,
	}

	missing := 0
	for _, rt := range routes {
		if skip[rt.raw] {
			continue
		}
		// /ws is registered for every method by one handler; the upgrade is
		// what the spec describes, and it is a GET.
		if rt.raw == "/ws" && rt.method != http.MethodGet {
			continue
		}
		key := strings.ToLower(rt.method) + " " + specPath(rt.raw)
		if !documented[key] {
			t.Errorf("%s %s is served but not in docs/openapi.yaml", rt.method, rt.raw)
			missing++
		}
	}
	if missing == 0 {
		t.Logf("%d routes, all documented", len(routes))
	}
}

// specPath turns a served path into the form openapi.yaml uses: without the
// /api/v1 prefix (it is the spec's server URL) and with parameter names
// reduced to {}, so a rename is not reported as a missing endpoint.
func specPath(p string) string {
	p = strings.TrimPrefix(p, "/api/v1")
	// chi's walk prints a mounted subrouter's index as "/chats/"; the spec
	// writes it as "/chats". The trailing slash is the same endpoint.
	if len(p) > 1 {
		p = strings.TrimSuffix(p, "/")
	}
	if p == "" {
		p = "/"
	}
	var b strings.Builder
	for {
		open := strings.Index(p, "{")
		if open < 0 {
			b.WriteString(p)
			return b.String()
		}
		closeAt := strings.Index(p[open:], "}")
		if closeAt < 0 {
			b.WriteString(p)
			return b.String()
		}
		b.WriteString(p[:open])
		b.WriteString("{}")
		p = p[open+closeAt+1:]
	}
}

// openapiOperations reads the spec and returns the set of "method path" it
// describes, in the same normalised form as specPath.
func openapiOperations(t *testing.T) map[string]bool {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("..", "..", "..", "docs", "openapi.yaml"))
	if err != nil {
		t.Fatalf("read openapi.yaml: %v", err)
	}
	var spec struct {
		Paths map[string]map[string]yaml.Node `yaml:"paths"`
	}
	if err := yaml.Unmarshal(raw, &spec); err != nil {
		t.Fatalf("parse openapi.yaml: %v", err)
	}
	if len(spec.Paths) < 100 {
		t.Fatalf("openapi.yaml describes only %d paths; the spec did not parse", len(spec.Paths))
	}

	methods := map[string]bool{"get": true, "post": true, "put": true, "patch": true, "delete": true, "head": true, "options": true}
	out := map[string]bool{}
	for path, ops := range spec.Paths {
		for method := range ops {
			if methods[strings.ToLower(method)] {
				out[strings.ToLower(method)+" "+specPath(path)] = true
			}
		}
	}
	return out
}
