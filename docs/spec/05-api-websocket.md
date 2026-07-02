# KISY Enterprise Technical Specification - Part 5 (REST API & WebSocket)

## API Principles

REST API under /api/v1. JSON only. UTC timestamps in ISO-8601. UUID
identifiers.\
Consistent error format: code, message, details, requestId. Versioning
required.\
Every endpoint protected by RBAC unless explicitly public.

## POST /auth/login

Authenticate user and create session.

## POST /auth/logout

Invalidate current session.

## POST /auth/logout-all

Invalidate every active session.

## POST /auth/register

Register using invitation token.

## POST /auth/refresh

Rotate refresh token.

## GET /users/me

Return current profile.

## PATCH /users/me

Update username/avatar/password.

## GET /groups

Return only visible groups.

## POST /groups

CEO/Admin creates group.

## GET /chats

List available chats.

## POST /messages

Send message.

## DELETE /messages/{id}

Delete message according to permissions.

## POST /files

Upload attachment.

## GET /audit

CEO audit log.

## POST /invites

Generate invitation token.

## WebSocket Events

Client-\>Server:\
auth, typing.start, typing.stop, message.send, reaction.add,
reaction.remove,\
presence.subscribe, read.confirmation.\
\
Server-\>Client:\
message.created, message.deleted, message.read, typing.started,\
typing.stopped, user.online, user.offline, notification.created,\
group.updated, role.changed, invite.used, audit.alert.

## API Specification Block 1

Define request schema, response schema, validation rules, RBAC
requirements, rate limits, idempotency, pagination, filtering, sorting,
audit events, error codes, OpenAPI examples and security requirements.

## API Specification Block 2

Define request schema, response schema, validation rules, RBAC
requirements, rate limits, idempotency, pagination, filtering, sorting,
audit events, error codes, OpenAPI examples and security requirements.

## API Specification Block 3

Define request schema, response schema, validation rules, RBAC
requirements, rate limits, idempotency, pagination, filtering, sorting,
audit events, error codes, OpenAPI examples and security requirements.

## API Specification Block 4

Define request schema, response schema, validation rules, RBAC
requirements, rate limits, idempotency, pagination, filtering, sorting,
audit events, error codes, OpenAPI examples and security requirements.

## API Specification Block 5

Define request schema, response schema, validation rules, RBAC
requirements, rate limits, idempotency, pagination, filtering, sorting,
audit events, error codes, OpenAPI examples and security requirements.

## API Specification Block 6

Define request schema, response schema, validation rules, RBAC
requirements, rate limits, idempotency, pagination, filtering, sorting,
audit events, error codes, OpenAPI examples and security requirements.

## API Specification Block 7

Define request schema, response schema, validation rules, RBAC
requirements, rate limits, idempotency, pagination, filtering, sorting,
audit events, error codes, OpenAPI examples and security requirements.

## API Specification Block 8

Define request schema, response schema, validation rules, RBAC
requirements, rate limits, idempotency, pagination, filtering, sorting,
audit events, error codes, OpenAPI examples and security requirements.

## API Specification Block 9

Define request schema, response schema, validation rules, RBAC
requirements, rate limits, idempotency, pagination, filtering, sorting,
audit events, error codes, OpenAPI examples and security requirements.

## API Specification Block 10

Define request schema, response schema, validation rules, RBAC
requirements, rate limits, idempotency, pagination, filtering, sorting,
audit events, error codes, OpenAPI examples and security requirements.

## API Specification Block 11

Define request schema, response schema, validation rules, RBAC
requirements, rate limits, idempotency, pagination, filtering, sorting,
audit events, error codes, OpenAPI examples and security requirements.

## API Specification Block 12

Define request schema, response schema, validation rules, RBAC
requirements, rate limits, idempotency, pagination, filtering, sorting,
audit events, error codes, OpenAPI examples and security requirements.

## API Specification Block 13

Define request schema, response schema, validation rules, RBAC
requirements, rate limits, idempotency, pagination, filtering, sorting,
audit events, error codes, OpenAPI examples and security requirements.

## API Specification Block 14

Define request schema, response schema, validation rules, RBAC
requirements, rate limits, idempotency, pagination, filtering, sorting,
audit events, error codes, OpenAPI examples and security requirements.

## API Specification Block 15

Define request schema, response schema, validation rules, RBAC
requirements, rate limits, idempotency, pagination, filtering, sorting,
audit events, error codes, OpenAPI examples and security requirements.

## API Specification Block 16

Define request schema, response schema, validation rules, RBAC
requirements, rate limits, idempotency, pagination, filtering, sorting,
audit events, error codes, OpenAPI examples and security requirements.

## API Specification Block 17

Define request schema, response schema, validation rules, RBAC
requirements, rate limits, idempotency, pagination, filtering, sorting,
audit events, error codes, OpenAPI examples and security requirements.

## API Specification Block 18

Define request schema, response schema, validation rules, RBAC
requirements, rate limits, idempotency, pagination, filtering, sorting,
audit events, error codes, OpenAPI examples and security requirements.

## API Specification Block 19

Define request schema, response schema, validation rules, RBAC
requirements, rate limits, idempotency, pagination, filtering, sorting,
audit events, error codes, OpenAPI examples and security requirements.

## API Specification Block 20

Define request schema, response schema, validation rules, RBAC
requirements, rate limits, idempotency, pagination, filtering, sorting,
audit events, error codes, OpenAPI examples and security requirements.
