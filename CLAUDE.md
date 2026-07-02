# KISY Enterprise Messenger — Инструкции для агента

## Что это
Полноценное production-ready корпоративное мессенджер-приложение.
Полная спецификация лежит в `docs/spec/` — прочитай ВСЕ файлы по порядку
(01–10) перед тем, как писать первую строку кода.

## Стек (обязателен, не менять)
- Frontend: React + TypeScript + Vite
- Backend: Go, Clean Architecture
- DB: PostgreSQL
- Cache: Redis
- Realtime: WebSockets
- Proxy: Nginx
- Deploy: Docker Compose
- Docs: OpenAPI

## Порядок работы (не пытайся сделать всё за один проход)
Работай поэтапно, каждый этап — рабочий и проверяемый кусок:

1. **Фундамент**: структура репозитория, docker-compose skeleton, БД-миграции (`docs/spec/04-database.md`)
2. **Auth и доступ**: инвайт-токены, JWT, ролевая модель 1–10 уровней (`01`, `06`)
3. **Backend core**: REST API по контрактам (`09-api-contracts.md`), WebSocket-слой (`05`)
4. **Business logic**: чаты, группы, права видимости между уровнями (`07`)
5. **Frontend**: UI по `02-frontend-ux.md` (Telegram-inspired, dark mode, glassmorphism)
6. **Security hardening**: пройтись отдельным проходом по `06-security.md` (CSP, rate limiting, аудит-логи, сканирование файлов)
7. **DevOps**: CI/CD, Docker, деплой-скрипты (`08-devops.md`)
8. **Тесты и документация**: unit/integration/e2e, OpenAPI, admin/dev guides

## Definition of Done (см. `10-master-prompt.md`)
- Нет заглушек, TODO, моков
- Нет ошибок компиляции
- Docker-деплой работает с чистого окружения
- Все эндпоинты задокументированы в OpenAPI
- API p95 < 200ms на типовых операциях

## Важно
- После каждого крупного этапа — запускай сборку и тесты, не копи технический долг
- Если что-то в спецификации неоднозначно — лучше спросить, чем додумывать
- Секреты (JWT-ключи, пароли БД и т.п.) — только через `.env`, никогда не хардкодить
