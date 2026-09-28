# База данных

PostgreSQL: локально 16, на проде Neon 18. Владелец схемы:
`backend/migrations` (формат golang-migrate,
`{version}_{name}.up.sql` / `.down.sql`), применяются бэкендом на старте
через `internal/platform/postgres`.

## Таблицы

Здесь их больше нет списком. Список был заморожен на июле (21 таблица из 53
миграций) и врал в трёх местах: обещал работающий RBAC по таблицам
`permissions`/`role_permissions` (права считаются по уровню, таблицы мертвы),
«редактирование отключено» (включено) и «файлы сканируются до показа»
(сканирования нет, есть чёрный список сигнатур — см. docs/security.md). Копия
схемы в отдельном файле неизбежно расходится с миграциями, поэтому её больше
нет (аудит C-15).

Актуальная схема — сами миграции:

```bash
ls backend/migrations/*.up.sql        # порядок и названия
grep -l "CREATE TABLE" backend/migrations/*.up.sql
```

Или из живой базы: `\dt` в psql.

## Соглашения

- Первичные ключи UUID (`gen_random_uuid()`, `pgcrypto`).
- Метки времени — `TIMESTAMPTZ`, всегда UTC.
- Удаление аккаунта: `users.deleted_at` + анонимизация (миграция 000050,
  `internal/users/deletion.go`), строка остаётся ради чужих сообщений. Ранее
  здесь было «аккаунты нельзя удалить, только деактивировать» — требование
  Google Play это изменило.
- `audit_logs` отклоняет `UPDATE`/`DELETE` на уровне триггера — неизменяем
  by design.
- `messages.chat_id` полиморфен (`private_chats.id` или `groups.id` в
  зависимости от `chat_type`); ссылочная целостность для него обеспечивается
  на уровне приложения, так как в Postgres нет нативного полиморфного FK.

## Запуск миграций

```bash
# из backend/
migrate -path migrations -database "$DATABASE_URL" up
migrate -path migrations -database "$DATABASE_URL" down 1
```

Бэкенд также применяет ожидающие миграции автоматически при старте в
непродакшн-окружениях (см. `internal/platform/postgres`).

## seeds/

Только фикстуры для разработки/демо (никогда структурные данные — иерархия
ролей поставляется миграцией, см. `backend/migrations/000012_seed_roles`).
