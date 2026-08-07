# Бесплатный деплой на Render

Разворачивает всё приложение (фронтенд + API + WebSocket + PostgreSQL +
Redis) на бесплатном тарифе Render одним blueprint'ом. Бэкенд сам отдаёт
собранный фронтенд, поэтому всё работает на одном домене (это важно для
cookie `SameSite=Strict`).

## Что понадобится

- Аккаунт **GitHub** (уже есть).
- Аккаунт **Render** (регистрация бесплатна, через GitHub — быстрее всего):
  https://render.com

## Шаг 1. Залить код на GitHub

Создай пустой репозиторий на GitHub (без README/лицензии), затем в корне
проекта:

```bash
git branch -M main
git remote add origin https://github.com/<ТВОЙ_ЛОГИН>/<РЕПО>.git
git push -u origin main
```

> `.env`, TLS-ключи и бэкапы не попадут в репозиторий — они в `.gitignore`.

## Шаг 2. Завести внешние БД и кэш (бесплатные, непропадающие)

Render'овские **free Postgres/Redis удаляются через 30 дней**, поэтому данные
держим на внешних сервисах, которые не истекают. Заводи их в **EU-регионе
(Frankfurt)** — рядом с сервисом, чтобы запросы backend→БД были быстрыми.

- **PostgreSQL → Neon** (neon.tech): создай проект в регионе *AWS Europe
  Central 1 (Frankfurt)*. Скопируй **прямую** connection-строку (сними
  галочку «Pooled connection» — pgx держит долгоживущий пул и prepared
  statements, а pgbouncer-пулер Neon их ломает). Это `DATABASE_URL`.
- **Redis → Upstash** (upstash.com): создай базу в регионе EU (Frankfurt/
  Germany). Возьми `rediss://…` строку. Это `REDIS_URL`.

## Шаг 3. Развернуть blueprint

1. Зайди на https://dashboard.render.com → **New +** → **Blueprint**.
2. Подключи свой GitHub-аккаунт и выбери запушенный репозиторий.
3. Render найдёт `render.yaml` и создаст **web-сервис `kisy` в Frankfurt**
   (БД и Redis он НЕ создаёт — они внешние).
4. Он попросит секреты, помеченные `sync:false` — впиши:
   - **`DATABASE_URL`** — строка от Neon (с `?sslmode=require`);
   - **`REDIS_URL`** — строка от Upstash;
   - **`BOOTSTRAP_CEO_PASSWORD`** — пароль первого входа CEO (≥ **12
     символов**, с буквой и цифрой, например `Kisy-Admin-2026`). Запомни.
5. Нажми **Apply**. Render соберёт образ (2–4 мин), применит миграции на
   пустой Neon-базе (создастся и аккаунт CEO) и запустит сервис.

Остальные секреты (`JWT_ACCESS_SECRET`, `JWT_REFRESH_SECRET`,
`IP_HASH_SALT`) Render сгенерирует сам.

## Шаг 4. Пользоваться

Адрес будет вида `https://kisy-XXXX.onrender.com`. Открой его и войди:

- **Логин:** `ceo`
- **Пароль:** тот, что задал на шаге 3.

Дальше приглашай пользователей (иконка щита → Приглашения), создавай чаты,
группы и доски задач.

## Особенности бесплатного тарифа

- **Засыпание.** Web-сервис уходит в сон после ~15 минут без запросов.
  Первый запрос после сна поднимает его за ~30–60 секунд (страница просто
  дольше грузится). WebSocket сам переподключится.
- **Срок жизни БД.** Данные на **Neon** (Postgres) и **Upstash** (Redis) —
  бесплатные тарифы там НЕ удаляются по таймеру, в отличие от Render'овских
  free-аддонов. Именно поэтому БД/кэш вынесены наружу (см. Шаг 2). Дамп на
  всякий случай: `pg_dump "$DATABASE_URL" | gzip > dump.sql.gz`.
- **Первый CEO.** Аккаунт создаётся только когда таблица пользователей
  пуста. После первого запуска можно убрать `BOOTSTRAP_CEO_PASSWORD` из
  переменных окружения и сменить пароль в профиле.

## Обновления

При `git push` в `main` Render автоматически пересоберёт и передеплоит
(`autoDeploy: true`).

## Локальная проверка образа (опционально)

Тот же all-in-one образ можно собрать и запустить локально:

```bash
docker build -t kisy .
docker run --rm -p 8080:8080 \
  -e DATABASE_URL=postgres://kisy:PASS@host:5432/kisy?sslmode=disable \
  -e REDIS_URL=redis://:PASS@host:6379 \
  -e JWT_ACCESS_SECRET=... -e JWT_REFRESH_SECRET=... \
  -e BOOTSTRAP_CEO_USERNAME=ceo -e BOOTSTRAP_CEO_PASSWORD=... \
  kisy
# приложение на http://localhost:8080
```
