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

## Статус

Этап 1 (Фундамент) — структура репозитория, миграции БД, docker-compose skeleton.
См. `CLAUDE.md` для порядка последующих этапов.
