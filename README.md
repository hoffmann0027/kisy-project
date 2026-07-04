# KISY Enterprise Messenger

Закрытый корпоративный мессенджер с инвайт-регистрацией, ролевой моделью
(10 уровней), приватными и групповыми чатами в реальном времени.

Полная спецификация: [docs/spec](docs/spec).

## Стек

| Слой       | Технология                          |
|------------|--------------------------------------|
| Frontend   | React + TypeScript + Vite            |
| Backend    | Go (Clean Architecture)              |
| DB         | PostgreSQL 16                        |
| Cache      | Redis                                |
| Realtime   | WebSockets                           |
| Proxy      | Nginx                                |
| Deploy     | Docker Compose                       |
| Docs       | OpenAPI                              |

## Структура репозитория

```
backend/    Go-сервис (Clean Architecture: domain/application/infrastructure/delivery)
frontend/   React + TS SPA (feature-sliced design)
database/   Обзор схемы, seed-данные
deploy/     Nginx, Docker вспомогательные файлы
docs/       Спецификация и документация
scripts/    Вспомогательные скрипты (миграции, бэкапы, разработка)
tests/      Интеграционные и e2e тесты
.github/    CI/CD workflows
```

## Быстрый старт (разработка)

```bash
cp .env.example .env
# отредактируйте .env: задайте реальные пароли и JWT-секреты
docker compose up --build
```

Всё доступно через единую точку входа Nginx: http://localhost — `/` отдаёт
frontend, `/api/*` и `/ws` проксируются на backend, `/health` — liveness
backend.

Для разработки без Docker: `cd backend && go run ./cmd/server` (нужны
локальные Postgres/Redis) и `cd frontend && npm run dev` (Vite на
http://localhost:5173, проксирует `/api` и `/ws` на `localhost:8080`).

## Операции и наблюдаемость

- `make help` — список задач (up/down/logs/test/lint/vuln/certs/backup…).
- CI: [.github/workflows/ci.yml](.github/workflows/ci.yml) — lint, тесты
  (+integration), govulncheck, npm audit, сборка образов. CD:
  [release.yml](.github/workflows/release.yml) — публикация в GHCR по тегу,
  деплой в staging и prod с ручным аппрувом. Полный гайд: [docs/devops.md](docs/devops.md).
- TLS 1.3 в проде: `make certs && make prod` (см. devops-гайд).
- Метрики Prometheus + Grafana: `make monitoring`
  (Prometheus :9090, Grafana :3000). Backend отдаёт метрики на `/metrics`
  (внутренний эндпоинт, не проксируется наружу).
- Бэкапы БД: `make backup` / `make restore`.

## Документация

- [docs/openapi.yaml](docs/openapi.yaml) — контракт REST API + WebSocket.
- [docs/security.md](docs/security.md) — модель угроз (STRIDE) и контроли.
- [docs/devops.md](docs/devops.md) — деплой, CI/CD, мониторинг, бэкапы.
- [docs/spec](docs/spec) — исходная спецификация.

## Статус

Реализованы этапы 1–7: фундамент, авторизация и доступ, backend core
(REST + WebSocket), бизнес-логика, фронтенд, security-hardening, DevOps/CI-CD.
Дополнительно: группы с ролевым доступом и Kanban-доски задач. См.
`CLAUDE.md` для порядка этапов; следующий — этап 8 (тесты и документация).
