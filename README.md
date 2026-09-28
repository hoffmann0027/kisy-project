# KISY Enterprise Messenger

Закрытый корпоративный мессенджер с ролевой моделью (10 уровней),
приватными и групповыми чатами в реальном времени, сообществами и лентой.

Аккаунтов два вида, и тип задаётся при регистрации: **пригласительный** (по
токену CEO, с уровнем 1–10) и **обычный** (без токена, вне иерархии уровней).
Второму недоступно всё, что на уровнях построено — рейтинг, условия
повышения, голосования, администрирование; чаты, группы, сообщества, звонки и
шифрование работают одинаково. Открытую регистрацию выключает
`REGISTRATION_OPEN=false`. Подробнее — «Два типа аккаунтов» в
[docs/admin-guide.md](docs/admin-guide.md).

Полная спецификация: [docs/spec](docs/spec).

## Стек

| Слой       | Технология                          |
|------------|--------------------------------------|
| Frontend   | React + TypeScript + Vite            |
| Backend    | Go, пакет на фичу (см. ниже)        |
| DB         | PostgreSQL (локально 16, прод — Neon 18) |
| Cache      | Redis (прод — Upstash)              |
| Realtime   | WebSockets                           |
| Proxy      | Nginx (в compose; на Render его нет) |
| Deploy     | Docker Compose или Render (`render.yaml`) |
| Docs       | OpenAPI                              |

## Структура репозитория

```
backend/    Go-сервис: пакет на фичу (internal/<фича>: домен, сервис, SQL, хендлер),
            композиционный корень — cmd/server/modules.go
frontend/   React + TS SPA (feature-sliced design, соблюдён примерно наполовину)
database/   Seed-данные; схема — источником истины служат backend/migrations
deploy/     Nginx, coturn, мониторинг, вспомогательные файлы Docker
docs/       Спецификация и документация
design/     Мастер-логотип и генератор иконок
scripts/    Вспомогательные скрипты (бэкапы, сертификаты, подпись Android)
tests/      Нагрузочные сценарии k6 (интеграционные тесты живут рядом с кодом,
            `*_integration_test.go` за тегом `integration`)
.github/    CI/CD workflows
```

Про «Clean Architecture» честно: слоёв domain/application/infrastructure нет —
хендлеры тонкие и без SQL, но сервисы работают с pgx напрямую. Так и осталось
осознанно; переписывание на книжные слои добавило бы шаблонного кода без
выигрыша (аудит E-07).

## Быстрый старт (разработка)

```bash
cp .env.example .env
# отредактируйте .env: задайте реальные пароли и JWT-секреты
docker compose up --build
```

Всё доступно через единую точку входа Nginx: http://localhost — `/` отдаёт
frontend, `/api/*` и `/ws` проксируются на backend, `/health` — liveness
backend.

Первый вход: логин `ceo`, пароль — `BOOTSTRAP_CEO_PASSWORD` из `.env`;
сразу после входа приложение потребует его сменить.

Для разработки без Docker бэкенду нужно окружение: `.env` он сам не читает, а
хосты по умолчанию — `postgres` и `redis` (имена сервисов compose). То есть
`go run ./cmd/server` запускается так:

```bash
cd backend
set -a; . ../.env; set +a          # bash; в PowerShell задайте переменные вручную
POSTGRES_HOST=localhost REDIS_HOST=localhost go run ./cmd/server
```

Фронтенд без Docker: `cd frontend && npm run dev` (Vite на
http://localhost:5173, проксирует `/api` и `/ws` на `localhost:8080`).

## Бесплатный хостинг (Render)

Всё приложение разворачивается на бесплатном тарифе Render одним blueprint'ом
(`render.yaml`): бэкенд сам отдаёт собранный фронтенд. База и Redis —
**внешние** управляемые сервисы (Postgres на Neon, Redis на Upstash): у самого
Render бесплатные Postgres/Redis удаляются через 30 дней, и это уже
произошло 2026-08-07. Пошаговая инструкция:
[docs/deploy-render.md](docs/deploy-render.md). Тот же all-in-one образ
(`Dockerfile` в корне) можно запустить локально.

## Операции и наблюдаемость

- `make help` — список задач (up/down/logs/test/lint/vuln/certs/backup…).
- CI: [.github/workflows/ci.yml](.github/workflows/ci.yml) — lint, тесты
  (+integration), govulncheck, npm audit, gitleaks, trivy, сборка образов.
  [release.yml](.github/workflows/release.yml) — публикация образов в GHCR по
  тегу; деплоя в нём нет, прод на Render деплоится сам из `main`. Полный
  гайд: [docs/devops.md](docs/devops.md).
- TLS 1.3 в проде: `make certs && make prod` (см. devops-гайд).
- Метрики Prometheus + Grafana: `make monitoring`
  (Prometheus :9090, Grafana :3000). Backend отдаёт метрики на `/metrics`; в
  production эндпоинт закрыт токеном `METRICS_TOKEN` (без него — 404), вне
  production открыт.
- Бэкапы БД: `make backup` / `make restore` — **только локальный compose**.
  Прод (Neon) бэкапится workflow «DB backup» и `scripts/db-backup.sh`;
  восстановление — `scripts/db-restore.sh`, порядок в
  [docs/runbook.md](docs/runbook.md), раздел «Backup / restore».

## Документация

- [docs/openapi.yaml](docs/openapi.yaml) — контракт REST API + WebSocket.
- [docs/admin-guide.md](docs/admin-guide.md) — руководство администратора (CEO).
- [docs/developer-guide.md](docs/developer-guide.md) — архитектура и разработка.
- [docs/security.md](docs/security.md) — модель угроз (STRIDE) и контроли.
- [docs/devops.md](docs/devops.md) — деплой, CI/CD, мониторинг, бэкапы.
- [docs/android.md](docs/android.md) — Android-приложение: сборка APK, push
  через Firebase, подписанный релиз для Google Play.
- [docs/mobile.md](docs/mobile.md) — адаптив под телефон.
- [docs/spec](docs/spec) — исходная спецификация.

## Статус

Все восемь этапов из `CLAUDE.md` пройдены. Сверх них: сообщества с лентой,
доски задач, календарь, заметки, опросы, рейтинг, 1:1 аудиозвонки (WebRTC +
coturn), сквозное шифрование личных чатов (MLS, RFC 9420), Android-приложение
на Capacitor с пушами через FCM, антиспам (Turnstile, лимиты по аккаунту,
карантин новых аккаунтов), удаление аккаунта, блокировки и жалобы.

Что известно и не закрыто: у групповых чатов и сообществ сквозного шифрования
нет (только TLS); сканирования файлов нет (есть чёрный список сигнатур);
наблюдаемости и алертов на проде нет; спецификация в `docs/spec` местами
расходится с кодом. Ближайшие шаги — публикация в Google Play
([docs/PLAY_LISTING.md](docs/PLAY_LISTING.md)).
