# KISY — Руководство разработчика

Как устроен код и как его расширять. Читайте вместе с
`docs/security.md`, `docs/devops.md` и `docs/openapi.yaml`.

## Стек и структура

Монорепозиторий: `backend/` (Go), `frontend/` (React + TS + Vite),
`deploy/` (Nginx, мониторинг), `docs/`, `scripts/`, `.github/`.

```
backend/
  cmd/server/          точка входа + композиционный корень (main.go, modules.go)
  internal/
    <domain>/          по пакету на модуль: domain.go, repository.go,
                       service.go, handler.go (+ *_test.go)
    platform/          сквозная инфраструктура: postgres, redis, logger, db,
                       ratelimit, security, metrics, testdb
  pkg/                 переиспользуемое, не зависящее от приложения
                       (httpresponse, httpjson, pagination)
  migrations/          SQL для golang-migrate (NNNNNN_name.up/.down.sql)
frontend/src/
  app/ pages/ widgets/ features/ entities/ shared/   (feature-sliced)
```

## Архитектура бэкенда

Чистое послойное разделение внутри каждого модуля:

- **domain.go** — сущности, DTO, sentinel-ошибки. Без I/O.
- **repository.go** — интерфейс `Repository` + реализация на Postgres.
  Каждый метод принимает `db.DBTX` (реализуется и `*pgxpool.Pool`, и
  `pgx.Tx`), поэтому вызов может выполняться сам по себе или внутри
  транзакции.
- **service.go** — сценарии использования, проверки прав, границы
  транзакций, аудит, публикация событий.
- **handler.go** — HTTP: декодирование/валидация, маппинг доменных ошибок
  на контракт API, никогда не раскрывает внутренние детали.

### Композиционный корень

`cmd/server/modules.go` — **единственное место**, где всё связывается.
Чтобы граф зависимостей оставался ацикличным, межмодульные зависимости
внедряются как **значения функций/интерфейсов**, а не через импорты.
Примеры:

- `chats` получает `ProfileLoader` и `UnreadLoader` (из users/readstate)
  вместо их импорта.
- `messages` получает `Authorizer{Private, Group}` и `ReactionLoader`.
- `boards` получает `Access{EnsureActorMember, IsFounder, IsMember}` на
  основе сервиса групп.
- WebSocket-хаб выступает `Publisher` для messages/reactions/boards.

Когда новому модулю нужен другой — предпочитайте внедрённую
функцию/интерфейс прямому импорту.

### Соглашения об ответах и ошибках

- Каждый ответ — это `pkg/httpresponse.Envelope`
  (`success/data/error/requestId/timestamp`).
- Правило сокрытия ресурсов: то, что выше допуска вызывающего, возвращает
  **тот же 404, что и несуществующий ресурс** — никогда 403 — чтобы нельзя
  было проверить существование.
- Никогда не раскрывайте текст внутренней ошибки; логируйте на сервере,
  возвращайте обобщённый код.

## Как добавить фичу (бэкенд)

1. Если нужно хранение — добавьте пару миграций в `backend/migrations`.
2. Создайте `internal/<feature>/` с domain/repository/service/handler.
3. Принимайте `db.DBTX` в методах репозитория; оборачивайте многошаговые
   записи в транзакцию.
4. Свяжите в `cmd/server/modules.go` и смонтируйте маршруты в `main.go`,
   за `RequireAuth` (и `RequireClearance`, где нужно).
5. Добавьте unit-тесты (чистая логика) и интеграционный тест с тегом
   `//go:build integration`, используя `internal/platform/testdb`.
6. Задокументируйте эндпоинты в `docs/openapi.yaml`.

## Архитектура фронтенда

Feature-sliced. `shared/api` содержит типизированный клиент, `endpoints.ts`
и `types.ts` (единый источник формы API). Серверное состояние живёт в
хуках TanStack Query под `entities/*/queries.ts`; чисто UI-состояние — в
Zustand-сторах (`shared/store`). WebSocket-клиент изолирован в `shared/ws`
и раздаётся в кэш запросов через `app/useRealtime.ts`. Стилизация — на
CSS-переменных из `shared/config/theme.css` (тёмный glassmorphism).

## Запуск и тесты

```bash
make up                 # весь стек на http://localhost
make test               # unit-тесты бэкенда (race)
make lint               # gofmt + vet + tsc
cd frontend && npm test # unit-тесты фронтенда (Vitest)

# интеграционные тесты (нужна БД):
cd backend
TEST_DATABASE_URL=postgres://kisy:<pw>@localhost:5432/kisy?sslmode=disable \
  go test -tags integration ./...
```

`TEST_DATABASE_URL` указывает на служебную БД; харнесс создаёт и удаляет
уникально названную базу для каждого теста, поэтому запуски изолированы и
воспроизводимы. CI выполняет те же команды (`.github/workflows/ci.yml`).

## Соглашения

- Go: чистый `gofmt`, чистый `go vet`, табличные тесты, обёрнутые ошибки
  (`fmt.Errorf("...: %w", err)`), sentinel-ошибки для управления потоком.
- TS: режим `strict`, без неиспользуемых локальных/параметров, алиасы путей
  (`@shared`, …).
- Миграции только добавляются и обратимы; никогда не редактируйте уже
  выпущенную.
- Секреты только через env/`.env`; никогда не хардкодить.
