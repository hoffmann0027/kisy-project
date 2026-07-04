# KISY DevOps & Deployment Guide

Covers local development, CI/CD, production deployment, observability,
backups and rollback (docs/spec/08-devops.md).

## Topology

Single Nginx edge → frontend static bundle + backend API/WebSocket.
Postgres and Redis are internal-only. Containers: `postgres`, `redis`,
`backend`, `frontend`, `nginx` (+ `prometheus`, `grafana` with the
monitoring overlay).

Compose files:

| File                          | Purpose                                        |
|-------------------------------|------------------------------------------------|
| `docker-compose.yml`          | Base stack (build from source, HTTP on :80)    |
| `docker-compose.override.yml` | Dev-only: publishes DB/Redis/backend on loopback |
| `docker-compose.prod.yml`     | TLS 1.3 edge + pulls published GHCR images     |
| `docker-compose.monitoring.yml` | Prometheus + Grafana                         |

## Local development

```bash
cp .env.example .env         # set real passwords + JWT secrets
make up                      # docker compose up -d --build
# app:      http://localhost
# health:   http://localhost/health
```

Without Docker: `cd backend && go run ./cmd/server` (needs local
Postgres/Redis) and `cd frontend && npm run dev`.

Common tasks: `make help`, `make test`, `make lint`, `make vuln`,
`make logs`, `make down`.

## CI (GitHub Actions — `.github/workflows/ci.yml`)

Runs on every push/PR to `main`/`master`:

1. **backend** — gofmt check, `go vet`, `go test -race`, integration tests
   (against ephemeral Postgres/Redis services), `govulncheck`, build.
2. **frontend** — `npm ci`, type-check, production build, `npm audit`.
3. **docker** — builds both images (no push) to validate the Dockerfiles.

Dependabot (`.github/dependabot.yml`) opens weekly grouped update PRs for
Go modules, npm, GitHub Actions and Docker base images.

## CD (GitHub Actions — `.github/workflows/release.yml`)

Triggered by a semver tag (`vX.Y.Z`):

1. **build-and-push** — builds `backend` and `frontend` images and pushes
   them to GHCR (`ghcr.io/<owner>/<repo>/{backend,frontend}`), tagged with
   the version and `latest`.
2. **deploy-staging** — SSH deploy to the staging host, then a `/ready`
   smoke check. Uses the `staging` GitHub Environment.
3. **deploy-production** — same, gated on the `production` Environment.
   Configure **required reviewers** on that Environment to make it a manual
   approval step.

Required repository secrets: `STAGING_HOST`, `PROD_HOST`, `DEPLOY_USER`,
`DEPLOY_SSH_KEY`. On each host, `/opt/kisy` holds the compose files and a
production `.env`.

### Production deploy (on the host)

```bash
scripts/gen-dev-certs.sh          # or install real CA certs into deploy/nginx/certs
export KISY_IMAGE_PREFIX=ghcr.io/<owner>/<repo>
export KISY_VERSION=v1.2.3
docker compose -f docker-compose.yml -f docker-compose.prod.yml pull
docker compose -f docker-compose.yml -f docker-compose.prod.yml up -d
```

Migrations are applied automatically on boot only outside production
(`APP_ENV != production`). In production, run them explicitly during the
deploy window (e.g. `make migrate-down` for rollback of a single step, or
the `migrate` CLI `up`).

### Rollback

Images are immutable per tag in GHCR, so a rollback is a redeploy of the
previous green tag: set `KISY_VERSION` to the older tag and
`docker compose ... up -d`, or re-run the Release workflow from that tag.

## Observability

```bash
make monitoring    # docker compose -f docker-compose.yml -f docker-compose.monitoring.yml up -d
```

- Backend exposes Prometheus metrics at `/metrics` (internal only — the
  edge proxy does not forward it). Metrics: `kisy_http_requests_total`,
  `kisy_http_request_duration_seconds` plus Go runtime metrics.
- Prometheus: http://localhost:9090 · Grafana: http://localhost:3000
  (admin / `GRAFANA_ADMIN_PASSWORD`), pre-provisioned with the Prometheus
  datasource.
- Health: `/health` (liveness) and `/ready` (DB + Redis readiness); both
  back the container healthchecks in compose.

## Backups & recovery

```bash
make backup                       # gzip pg_dump into backups/, 14-file retention
BACKUP=backups/kisy-<stamp>.sql.gz make restore
```

Schedule `scripts/backup.sh` nightly via cron; set `BACKUP_GPG_RECIPIENT`
to encrypt at rest and ship the artifact off-site. `scripts/restore.sh`
restores the newest (or a named) dump.

## Secrets

All secrets come from environment / `.env`, never committed. `.env`, TLS
private keys (`deploy/nginx/certs/`) and `backups/` are git-ignored.
