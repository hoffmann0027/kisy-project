# KISY developer & operations tasks. Requires Docker, Go and Node for the
# local targets. Run `make help` for the list.
.DEFAULT_GOAL := help
COMPOSE := docker compose
PROD := $(COMPOSE) -f docker-compose.yml -f docker-compose.prod.yml
MON := $(COMPOSE) -f docker-compose.yml -f docker-compose.monitoring.yml

.PHONY: help up down logs ps rebuild \
        backend-test backend-lint frontend-build frontend-lint test lint \
        vuln certs prod monitoring backup restore migrate-down

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | \
	  awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-16s\033[0m %s\n", $$1, $$2}'

up: ## Start the full stack (build if needed)
	$(COMPOSE) up -d --build

down: ## Stop the stack
	$(COMPOSE) down

logs: ## Tail all service logs
	$(COMPOSE) logs -f

ps: ## Show container status
	$(COMPOSE) ps

rebuild: ## Rebuild and restart backend + frontend
	$(COMPOSE) up -d --build backend frontend

backend-lint: ## gofmt + go vet
	cd backend && gofmt -l . && go vet ./...

backend-test: ## Run Go unit tests (race)
	cd backend && go test -race ./...

vuln: ## Run govulncheck + npm audit (shipped deps)
	cd backend && go run golang.org/x/vuln/cmd/govulncheck@latest ./...
	cd frontend && npm audit --omit=dev --audit-level=high

frontend-lint: ## TypeScript type-check
	cd frontend && npm run typecheck

frontend-build: ## Production build of the SPA
	cd frontend && npm ci && npm run build

lint: backend-lint frontend-lint ## Lint everything

test: backend-test ## Run the test suites

certs: ## Generate self-signed dev TLS certificates
	bash scripts/gen-dev-certs.sh

prod: ## Start the TLS production stack (needs certs)
	$(PROD) up -d --build

monitoring: ## Start the stack with Prometheus + Grafana
	$(MON) up -d --build

backup: ## Dump the database to backups/
	bash scripts/backup.sh

restore: ## Restore the newest backup (BACKUP=path to override)
	bash scripts/restore.sh $(BACKUP)

migrate-down: ## Roll back the last migration against the running DB (dev only)
	$(COMPOSE) run --rm -v $(CURDIR)/backend/migrations:/m migrate/migrate \
	  -path=/m -database "postgres://$(POSTGRES_USER):$(POSTGRES_PASSWORD)@postgres:5432/$(POSTGRES_DB)?sslmode=disable" down 1
