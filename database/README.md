# База данных

PostgreSQL 16. Владелец схемы: `backend/migrations` (формат golang-migrate,
`{version}_{name}.up.sql` / `.down.sql`), применяются бэкендом на старте
через `internal/platform/postgres`.

## Таблицы

| Таблица             | Назначение                                          |
|---------------------|------------------------------------------------------|
| roles                | Фиксированная иерархия допуска из 10 уровней          |
| permissions          | Каталог гранулярных разрешений                       |
| role_permissions     | Связь роль → разрешение                              |
| users                | Аккаунты, профиль, учётные данные                    |
| sessions             | Refresh-token сессии по устройствам                   |
| invitation_tokens    | Токены регистрации от CEO, TTL 120с, одноразовые      |
| groups               | Метаданные группы, минимальный уровень допуска        |
| group_members        | Членство в группах                                    |
| private_chats        | Личные диалоги один-на-один                            |
| messages             | Сообщения чата (редактирование отключено by design)    |
| attachments          | Файлы к сообщениям, сканируются до показа               |
| reactions            | Эмодзи-реакции на сообщения                            |
| message_mentions     | @упоминания внутри текста сообщения                    |
| audit_logs           | Append-only журнал каждого привилегированного действия  |
| notifications        | Непрочитанные уведомления по пользователю               |
| favorites            | Закреплённые/избранные чаты                             |
| chat_read_state      | Позиция прочтения по пользователю (счётчики непрочит.)  |
| boards               | Доска задач на группу (Trello-style)                    |
| board_columns        | Колонки доски                                           |
| board_cards          | Карточки доски (исполнитель, метка, срок, позиция)      |
| search_index         | Полнотекстовый поиск (`tsvector`, GIN-индекс)            |

## Соглашения

- Первичные ключи UUID (`gen_random_uuid()`, `pgcrypto`).
- Метки времени — `TIMESTAMPTZ`, всегда UTC.
- Нет жёсткого удаления `users` (аккаунты нельзя удалить, только
  деактивировать).
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
