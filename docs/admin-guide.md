# KISY Administrator Guide

For the **CEO** (clearance level 1), the only account with unrestricted
access. Everything here maps to the Admin panel (shield icon) and the
`/admin` API.

## The clearance model

Ten levels, 1 (CEO) strongest … 10 (Guest) weakest. A **lower number means
higher clearance**. Visibility flows downward:

- A group has a **minimum clearance** (`minRoleLevel`). It is visible only
  to users at that level or **stronger**. Weaker users cannot see it — and
  the API returns *not found*, never *forbidden*, so they cannot even infer
  it exists.
- Higher clearance may start a private chat with same-or-lower users; a
  weaker user cannot initiate a conversation "upward".

## Onboarding a user (invitations)

Registration is impossible without an invitation.

1. Admin panel → **Приглашения** → *Создать приглашение*.
2. Copy the **token** or the ready **registration link**.
3. Send it to the new employee. The token is valid **exactly 120 seconds**
   and works **once**. If it expires, generate a new one.

The new account starts at the lowest clearance (level 10). Promote it as
needed (below).

## Managing users

Admin panel → **Пользователи**:

- **Change role** — pick a new level in the dropdown. You cannot change
  your own role (prevents locking the last CEO out).
- **Reset password** — set a new password; this **ends all of that user's
  sessions** and forces a change on next login. Use for lockouts or lost
  credentials.
- **Deactivate / Activate** — deactivating blocks login and ends the
  user's sessions immediately. You cannot deactivate yourself. Accounts are
  never deleted, only deactivated.

## Groups & task boards

- Any user may create a group, but its minimum clearance **cannot exceed
  their own** — they must be able to belong to it.
- **Deleting a group:** the **CEO may delete any group**; otherwise only
  the group's **founder** may delete theirs. Deletion removes the group's
  chat, members and task board.
- Each group can have a **Trello-style task board**. Only the founder
  creates the board and manages its columns; any member adds and moves
  cards.

## Audit log

Admin panel → **Аудит** (API: `GET /admin/audit`). Every privileged action
is recorded immutably: logins and failures, lockouts, invitation
create/use, registrations, role changes, password resets,
activate/deactivate, group create/delete, message deletions and refresh
token-reuse security events. Each row carries the actor, target, hashed IP,
session and request id, and a UTC timestamp. The log **cannot be edited or
deleted** — the database rejects any such attempt.

## Security notes

- Passwords are Argon2id-hashed; the system never stores or shows them.
- Five failed logins lock an account for 15 minutes; rate limits also apply
  per IP.
- Sessions use HTTPOnly, SameSite=Strict cookies; "log out everywhere"
  revokes every device.

## Backups

Operators run `make backup` (nightly via cron in production) to dump the
database, and `make restore` to recover. See `docs/devops.md`.

## First-run bootstrap

The very first CEO account is created from `BOOTSTRAP_CEO_USERNAME` /
`BOOTSTRAP_CEO_PASSWORD` in `.env` when the database is empty. Remove those
variables after the first successful start and change the password from the
profile screen.
