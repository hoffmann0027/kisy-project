# Database

PostgreSQL 16. Schema owner: `backend/migrations` (golang-migrate format,
`{version}_{name}.up.sql` / `.down.sql`), applied by the backend on startup
via `internal/platform/postgres`.

## Tables

| Table               | Purpose                                             |
|---------------------|------------------------------------------------------|
| roles                | Fixed 10-level clearance hierarchy                   |
| permissions          | Fine-grained permission catalog                      |
| role_permissions     | Role → permission mapping                            |
| users                | Accounts, profile, credentials                       |
| sessions             | Refresh-token sessions per device                     |
| invitation_tokens    | CEO-issued, 120s TTL, single-use registration tokens  |
| groups               | Group metadata, minimum clearance level               |
| group_members        | Group membership                                      |
| private_chats        | One-to-one conversations                               |
| messages             | Chat messages (editing disabled by design)             |
| attachments          | Files linked to messages, scanned before visibility    |
| reactions            | Emoji reactions on messages                            |
| message_mentions     | @mentions inside message text                          |
| audit_logs           | Append-only log of every privileged action              |
| notifications        | Per-user unread notifications                          |
| favorites            | Pinned/favorite chats                                   |
| search_index         | Full-text search (`tsvector`, GIN-indexed)               |

## Conventions

- UUID primary keys (`gen_random_uuid()`, `pgcrypto`).
- Timestamps are `TIMESTAMPTZ`, always UTC.
- No hard deletes on `users` (accounts cannot be deleted, only deactivated).
- `audit_logs` rejects `UPDATE`/`DELETE` at the trigger level — immutable by design.
- `messages.chat_id` is polymorphic (`private_chats.id` or `groups.id`
  depending on `chat_type`); referential integrity for it is enforced in
  the application layer since Postgres has no native polymorphic FK.

## Running migrations

```bash
# from backend/
migrate -path migrations -database "$DATABASE_URL" up
migrate -path migrations -database "$DATABASE_URL" down 1
```

The backend also applies pending migrations automatically on boot in
non-production environments (see `internal/platform/postgres`).

## seeds/

Development/demo fixtures only (never structural data — the role
hierarchy ships as a migration, see `backend/migrations/000012_seed_roles`).
