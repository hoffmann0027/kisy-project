# KISY Developer Guide

How the codebase is organized and how to extend it. Pair this with
`docs/security.md`, `docs/devops.md` and `docs/openapi.yaml`.

## Stack & layout

Monorepo: `backend/` (Go), `frontend/` (React + TS + Vite), `deploy/`
(Nginx, monitoring), `docs/`, `scripts/`, `.github/`.

```
backend/
  cmd/server/          entrypoint + composition root (main.go, modules.go)
  internal/
    <domain>/          one package per module: domain.go, repository.go,
                       service.go, handler.go (+ *_test.go)
    platform/          cross-cutting infra: postgres, redis, logger, db,
                       ratelimit, security, metrics, testdb
  pkg/                 reusable, app-agnostic (httpresponse, httpjson,
                       pagination)
  migrations/          golang-migrate SQL (NNNNNN_name.up/.down.sql)
frontend/src/
  app/ pages/ widgets/ features/ entities/ shared/   (feature-sliced)
```

## Backend architecture

Clean-ish layering per module:

- **domain.go** — entities, DTOs, sentinel errors. No I/O.
- **repository.go** — a `Repository` interface + a Postgres implementation.
  Every method takes a `db.DBTX` (satisfied by both `*pgxpool.Pool` and
  `pgx.Tx`) so a call can run standalone or inside a transaction.
- **service.go** — use-cases, permission checks, transaction boundaries,
  auditing, publishing.
- **handler.go** — HTTP: decode/validate, map domain errors to the API
  contract, never leak internals.

### The composition root

`cmd/server/modules.go` is the **one place** that wires everything. To keep
the dependency graph acyclic, cross-module dependencies are injected as
**function/interface values**, not imports. Examples:

- `chats` gets a `ProfileLoader` and `UnreadLoader` (from users/readstate)
  instead of importing them.
- `messages` gets an `Authorizer{Private, Group}` and a `ReactionLoader`.
- `boards` gets an `Access{EnsureActorMember, IsFounder, IsMember}` backed
  by the groups service.
- The WebSocket hub is the `Publisher` for messages/reactions/boards.

When a new module needs another, prefer an injected func/interface over a
direct import.

### Response & error conventions

- Every response is a `pkg/httpresponse.Envelope`
  (`success/data/error/requestId/timestamp`).
- Hidden-resource rule: something above the caller's clearance returns the
  **same 404 as a nonexistent resource** — never 403 — so existence cannot
  be probed.
- Never surface internal error text; log server-side, return a generic code.

## Adding a feature (backend)

1. If it needs storage, add a migration pair in `backend/migrations`.
2. Create `internal/<feature>/` with domain/repository/service/handler.
3. Take a `db.DBTX` in repository methods; wrap multi-step writes in a tx.
4. Wire it in `cmd/server/modules.go` and mount routes in `main.go`, behind
   `RequireAuth` (and `RequireClearance` where needed).
5. Add unit tests (pure logic) and an `//go:build integration` test using
   `internal/platform/testdb`.
6. Document endpoints in `docs/openapi.yaml`.

## Frontend architecture

Feature-sliced. `shared/api` holds the typed client, `endpoints.ts` and
`types.ts` (the single source of API shape). Server state lives in TanStack
Query hooks under `entities/*/queries.ts`; UI-only state in Zustand stores
(`shared/store`). The WebSocket client is isolated in `shared/ws` and routed
into the query cache by `app/useRealtime.ts`. Styling uses CSS variables
from `shared/config/theme.css` (dark glassmorphism).

## Running & testing

```bash
make up                 # full stack at http://localhost
make test               # backend unit tests (race)
make lint               # gofmt + vet + tsc
cd frontend && npm test # frontend unit tests (Vitest)

# integration tests (need a database):
cd backend
TEST_DATABASE_URL=postgres://kisy:<pw>@localhost:5432/kisy?sslmode=disable \
  go test -tags integration ./...
```

`TEST_DATABASE_URL` points at a maintenance DB; the harness creates and
drops a uniquely-named database per test, so runs are isolated and
repeatable. CI runs the same commands (`.github/workflows/ci.yml`).

## Conventions

- Go: `gofmt` clean, `go vet` clean, table-driven tests, wrapped errors
  (`fmt.Errorf("...: %w", err)`), sentinel errors for control flow.
- TS: `strict` mode, no unused locals/params, path aliases (`@shared`, …).
- Migrations are append-only and reversible; never edit a shipped one.
- Secrets only via env/`.env`; never hardcode.
