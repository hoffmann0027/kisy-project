# KISY — DevOps и руководство по развёртыванию

Охватывает локальную разработку, CI/CD, продакшн-развёртывание,
наблюдаемость, резервные копии и откат (`docs/spec/08-devops.md`).

## Где что работает

**Прод — Render** (регион Frankfurt), один web-сервис из корневого
`Dockerfile`: тот же процесс отдаёт и API, и собранный SPA (`WEB_DIR`),
Nginx впереди нет. Деплой автоматический на каждый push в `main`
(`render.yaml`, `autoDeploy: true`) — отдельного шага в CI нет. Данные
живут во внешних управляемых сервисах: **Postgres — Neon**, **Redis —
Upstash** (бесплатные Postgres/Redis самого Render удаляются через 30 дней,
это уже случилось 2026-08-07). Пошагово — [deploy-render.md](deploy-render.md).

Откат прода: в дашборде Render *Manual Deploy* → выбрать более ранний коммит.

**Самостоятельный хостинг (compose)** — вариант ниже: единый edge-Nginx и
образы из GHCR. Для него и написаны остальные разделы этого документа.

## Топология (compose)

Единый edge-Nginx → статический бандл фронтенда + API/WebSocket бэкенда.
Postgres и Redis только внутренние. Контейнеры: `postgres`, `redis`,
`backend`, `frontend`, `nginx` (+ `prometheus`, `grafana` при оверлее
мониторинга).

Compose-файлы:

| Файл                          | Назначение                                     |
|-------------------------------|------------------------------------------------|
| `docker-compose.yml`          | Базовый стек (сборка из исходников, HTTP на :80) |
| `docker-compose.override.yml` | Только dev: публикует БД/Redis/backend на loopback |
| `docker-compose.prod.yml`     | Edge с TLS 1.3 + тянет опубликованные образы GHCR |
| `docker-compose.monitoring.yml` | Prometheus + Grafana                         |

## Локальная разработка

```bash
cp .env.example .env         # задайте реальные пароли + JWT-секреты
make up                      # docker compose up -d --build
# приложение: http://localhost
# health:     http://localhost/health
```

Без Docker: `cd backend && go run ./cmd/server` (нужны локальные
Postgres/Redis) и `cd frontend && npm run dev`.

Частые задачи: `make help`, `make test`, `make lint`, `make vuln`,
`make logs`, `make down`.

## CI (GitHub Actions — `.github/workflows/ci.yml`)

Запускается на каждый push/PR в `main`/`master`:

1. **backend** — проверка gofmt, `go vet`, `go test -race`, интеграционные
   тесты (против эфемерных сервисов Postgres/Redis), `govulncheck`, сборка.
2. **frontend** — `npm ci`, type-check, unit-тесты (Vitest),
   продакшн-сборка, `npm audit` (shipped-зависимости, `--omit=dev`).
3. **docker** — собирает оба образа (без push) для проверки Dockerfile.

Dependabot (`.github/dependabot.yml`) еженедельно открывает сгруппированные
PR с обновлениями Go-модулей, npm, GitHub Actions и базовых Docker-образов.

## Публикация образов (GitHub Actions — `.github/workflows/release.yml`)

Триггер — semver-тег (`vX.Y.Z`): собирает образы `backend` и `frontend` и
пушит их в GHCR (`ghcr.io/<owner>/<repo>/{backend,frontend}`) с тегом версии
и `latest`.

Деплоя в этом workflow нет. Раньше здесь были два SSH-шага на
`staging.kisy.example` и `kisy.example` — хосты, которых никогда не
существовало, так что первый же тег упал бы на них (аудит C-13). Прод
деплоится сам, из `main`, силами Render.

### Деплой self-hosted стенда (на своём хосте)

```bash
scripts/gen-dev-certs.sh          # или установите реальные CA-сертификаты в deploy/nginx/certs
export KISY_IMAGE_PREFIX=ghcr.io/<owner>/<repo>
export KISY_VERSION=v1.2.3
docker compose -f docker-compose.yml -f docker-compose.prod.yml pull
docker compose -f docker-compose.yml -f docker-compose.prod.yml up -d
```

Миграции применяются автоматически на старте только вне продакшна
(`APP_ENV != production`). В проде запускайте их явно в окно деплоя
(например `make migrate-down` для отката одного шага или `migrate` CLI
`up`).

### Откат

Образы неизменяемы для каждого тега в GHCR, поэтому откат — это повторный
деплой предыдущего «зелёного» тега: задайте `KISY_VERSION` на более старый
тег и `docker compose ... up -d`, либо перезапустите Release-workflow с того
тега.

## Наблюдаемость

```bash
make monitoring    # docker compose -f docker-compose.yml -f docker-compose.monitoring.yml up -d
```

- Бэкенд отдаёт метрики Prometheus на `/metrics` (только внутренний
  эндпоинт — edge не проксирует его наружу). Метрики:
  `kisy_http_requests_total`, `kisy_http_request_duration_seconds` плюс
  runtime-метрики Go.
- Prometheus: http://localhost:9090 · Grafana: http://localhost:3000
  (admin / `GRAFANA_ADMIN_PASSWORD`), с преднастроенным источником данных
  Prometheus.
- Health: `/health` (liveness) и `/ready` (готовность БД + Redis); оба
  используются в healthcheck контейнеров в compose.

## Резервные копии и восстановление

**Прод — не здесь.** База живёт на Neon, и до неё `docker compose exec` не
достаёт. Прод бэкапит workflow «DB backup» (`scripts/db-backup.sh`,
шифрование обязательно, артефакт в приватный бакет), восстанавливает
`scripts/db-restore.sh`. Порядок действий и нужные секреты —
[runbook.md](runbook.md), раздел «Backup / restore».

**Локальный compose:**

```bash
make backup                       # gzip pg_dump в backups/, хранение 14 файлов
BACKUP=backups/kisy-<stamp>.sql.gz make restore
```

`BACKUP_GPG_RECIPIENT` шифрует дамп (тогда файл будет `.sql.gz.gpg`, и
`restore.sh` расшифрует его сам). Восстановление идёт с
`ON_ERROR_STOP=1`: дамп поверх непустой базы обязан падать, а не
сообщать об успехе, применившись наполовину (аудит C-01).

## Секреты

Все секреты берутся из окружения / `.env`, никогда не коммитятся. `.env`,
приватные TLS-ключи (`deploy/nginx/certs/`) и `backups/` в `.gitignore`.
