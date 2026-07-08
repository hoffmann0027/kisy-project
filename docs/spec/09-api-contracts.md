# KISY Enterprise Technical Specification - Part 9 (Detailed API & Data Contracts)

## Response Format

Every response includes success, data, error, requestId, timestamp.

## Message Object

id(UUID), chatId, senderId, text, attachments\[\], reactions\[\],
createdAt, deletedAt, replyTo, mentions\[\].

## User Object

id, username, roleLevel, avatarUrl, status, lastSeen, createdAt.

## Invitation Object

token, creatorId, expiresAt, usedAt, usedBy.

## Call Objects (voice calls)

CallLog: id, direction(incoming|outgoing), status(completed|missed|rejected|
canceled|failed), peer{ id, displayName, avatarUrl }, chatId, startedAt,
answeredAt, endedAt, durationSeconds.

IceConfig: iceServers[] where each entry is { urls[], username?, credential? }.

## Call Endpoints

GET /calls/ice-config — RTCConfiguration for WebRTC: STUN plus, when TURN is
configured, a TURN entry with short-lived HMAC credentials derived from the
coturn shared secret (the secret never leaves the server). Rate-limited.

GET /calls/history?limit&offset — the caller's call journal, newest first,
mapped to their perspective (direction + the other party). limit 1..100
(default 50).

Live call signaling (call.invite/answer/ice/reject/cancel/hangup →
call.incoming/answered/ice/rejected/canceled/ended/busy/timeout) runs over the
WebSocket gateway; see Part 5. Media is peer-to-peer (WebRTC).

## Pagination

Cursor-based pagination. Stable ordering. Default limit 50. Maximum 200.

## Error Codes

AUTH_INVALID_TOKEN, AUTH_EXPIRED, ACCESS_DENIED, RESOURCE_NOT_FOUND,
RATE_LIMITED, VALIDATION_FAILED, INTERNAL_ERROR.

## Endpoint Contract 1

Define HTTP method, URI, request headers, authentication, JSON request
schema, validation rules, successful response example, error responses,
RBAC matrix, audit events, websocket side-effects, performance target
(\<200ms p95 where applicable), idempotency requirements and OpenAPI
documentation.

## Endpoint Contract 2

Define HTTP method, URI, request headers, authentication, JSON request
schema, validation rules, successful response example, error responses,
RBAC matrix, audit events, websocket side-effects, performance target
(\<200ms p95 where applicable), idempotency requirements and OpenAPI
documentation.

## Endpoint Contract 3

Define HTTP method, URI, request headers, authentication, JSON request
schema, validation rules, successful response example, error responses,
RBAC matrix, audit events, websocket side-effects, performance target
(\<200ms p95 where applicable), idempotency requirements and OpenAPI
documentation.

## Endpoint Contract 4

Define HTTP method, URI, request headers, authentication, JSON request
schema, validation rules, successful response example, error responses,
RBAC matrix, audit events, websocket side-effects, performance target
(\<200ms p95 where applicable), idempotency requirements and OpenAPI
documentation.

## Endpoint Contract 5

Define HTTP method, URI, request headers, authentication, JSON request
schema, validation rules, successful response example, error responses,
RBAC matrix, audit events, websocket side-effects, performance target
(\<200ms p95 where applicable), idempotency requirements and OpenAPI
documentation.

## Endpoint Contract 6

Define HTTP method, URI, request headers, authentication, JSON request
schema, validation rules, successful response example, error responses,
RBAC matrix, audit events, websocket side-effects, performance target
(\<200ms p95 where applicable), idempotency requirements and OpenAPI
documentation.

## Endpoint Contract 7

Define HTTP method, URI, request headers, authentication, JSON request
schema, validation rules, successful response example, error responses,
RBAC matrix, audit events, websocket side-effects, performance target
(\<200ms p95 where applicable), idempotency requirements and OpenAPI
documentation.

## Endpoint Contract 8

Define HTTP method, URI, request headers, authentication, JSON request
schema, validation rules, successful response example, error responses,
RBAC matrix, audit events, websocket side-effects, performance target
(\<200ms p95 where applicable), idempotency requirements and OpenAPI
documentation.

## Endpoint Contract 9

Define HTTP method, URI, request headers, authentication, JSON request
schema, validation rules, successful response example, error responses,
RBAC matrix, audit events, websocket side-effects, performance target
(\<200ms p95 where applicable), idempotency requirements and OpenAPI
documentation.

## Endpoint Contract 10

Define HTTP method, URI, request headers, authentication, JSON request
schema, validation rules, successful response example, error responses,
RBAC matrix, audit events, websocket side-effects, performance target
(\<200ms p95 where applicable), idempotency requirements and OpenAPI
documentation.

## Endpoint Contract 11

Define HTTP method, URI, request headers, authentication, JSON request
schema, validation rules, successful response example, error responses,
RBAC matrix, audit events, websocket side-effects, performance target
(\<200ms p95 where applicable), idempotency requirements and OpenAPI
documentation.

## Endpoint Contract 12

Define HTTP method, URI, request headers, authentication, JSON request
schema, validation rules, successful response example, error responses,
RBAC matrix, audit events, websocket side-effects, performance target
(\<200ms p95 where applicable), idempotency requirements and OpenAPI
documentation.

## Endpoint Contract 13

Define HTTP method, URI, request headers, authentication, JSON request
schema, validation rules, successful response example, error responses,
RBAC matrix, audit events, websocket side-effects, performance target
(\<200ms p95 where applicable), idempotency requirements and OpenAPI
documentation.

## Endpoint Contract 14

Define HTTP method, URI, request headers, authentication, JSON request
schema, validation rules, successful response example, error responses,
RBAC matrix, audit events, websocket side-effects, performance target
(\<200ms p95 where applicable), idempotency requirements and OpenAPI
documentation.

## Endpoint Contract 15

Define HTTP method, URI, request headers, authentication, JSON request
schema, validation rules, successful response example, error responses,
RBAC matrix, audit events, websocket side-effects, performance target
(\<200ms p95 where applicable), idempotency requirements and OpenAPI
documentation.

## Endpoint Contract 16

Define HTTP method, URI, request headers, authentication, JSON request
schema, validation rules, successful response example, error responses,
RBAC matrix, audit events, websocket side-effects, performance target
(\<200ms p95 where applicable), idempotency requirements and OpenAPI
documentation.

## Endpoint Contract 17

Define HTTP method, URI, request headers, authentication, JSON request
schema, validation rules, successful response example, error responses,
RBAC matrix, audit events, websocket side-effects, performance target
(\<200ms p95 where applicable), idempotency requirements and OpenAPI
documentation.

## Endpoint Contract 18

Define HTTP method, URI, request headers, authentication, JSON request
schema, validation rules, successful response example, error responses,
RBAC matrix, audit events, websocket side-effects, performance target
(\<200ms p95 where applicable), idempotency requirements and OpenAPI
documentation.

## Endpoint Contract 19

Define HTTP method, URI, request headers, authentication, JSON request
schema, validation rules, successful response example, error responses,
RBAC matrix, audit events, websocket side-effects, performance target
(\<200ms p95 where applicable), idempotency requirements and OpenAPI
documentation.

## Endpoint Contract 20

Define HTTP method, URI, request headers, authentication, JSON request
schema, validation rules, successful response example, error responses,
RBAC matrix, audit events, websocket side-effects, performance target
(\<200ms p95 where applicable), idempotency requirements and OpenAPI
documentation.

## Endpoint Contract 21

Define HTTP method, URI, request headers, authentication, JSON request
schema, validation rules, successful response example, error responses,
RBAC matrix, audit events, websocket side-effects, performance target
(\<200ms p95 where applicable), idempotency requirements and OpenAPI
documentation.

## Endpoint Contract 22

Define HTTP method, URI, request headers, authentication, JSON request
schema, validation rules, successful response example, error responses,
RBAC matrix, audit events, websocket side-effects, performance target
(\<200ms p95 where applicable), idempotency requirements and OpenAPI
documentation.

## Endpoint Contract 23

Define HTTP method, URI, request headers, authentication, JSON request
schema, validation rules, successful response example, error responses,
RBAC matrix, audit events, websocket side-effects, performance target
(\<200ms p95 where applicable), idempotency requirements and OpenAPI
documentation.

## Endpoint Contract 24

Define HTTP method, URI, request headers, authentication, JSON request
schema, validation rules, successful response example, error responses,
RBAC matrix, audit events, websocket side-effects, performance target
(\<200ms p95 where applicable), idempotency requirements and OpenAPI
documentation.

## Endpoint Contract 25

Define HTTP method, URI, request headers, authentication, JSON request
schema, validation rules, successful response example, error responses,
RBAC matrix, audit events, websocket side-effects, performance target
(\<200ms p95 where applicable), idempotency requirements and OpenAPI
documentation.

## Endpoint Contract 26

Define HTTP method, URI, request headers, authentication, JSON request
schema, validation rules, successful response example, error responses,
RBAC matrix, audit events, websocket side-effects, performance target
(\<200ms p95 where applicable), idempotency requirements and OpenAPI
documentation.

## Endpoint Contract 27

Define HTTP method, URI, request headers, authentication, JSON request
schema, validation rules, successful response example, error responses,
RBAC matrix, audit events, websocket side-effects, performance target
(\<200ms p95 where applicable), idempotency requirements and OpenAPI
documentation.

## Endpoint Contract 28

Define HTTP method, URI, request headers, authentication, JSON request
schema, validation rules, successful response example, error responses,
RBAC matrix, audit events, websocket side-effects, performance target
(\<200ms p95 where applicable), idempotency requirements and OpenAPI
documentation.

## Endpoint Contract 29

Define HTTP method, URI, request headers, authentication, JSON request
schema, validation rules, successful response example, error responses,
RBAC matrix, audit events, websocket side-effects, performance target
(\<200ms p95 where applicable), idempotency requirements and OpenAPI
documentation.

## Endpoint Contract 30

Define HTTP method, URI, request headers, authentication, JSON request
schema, validation rules, successful response example, error responses,
RBAC matrix, audit events, websocket side-effects, performance target
(\<200ms p95 where applicable), idempotency requirements and OpenAPI
documentation.
