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

## Attachment Object (stage A: media metadata + chunked upload)

id(UUID), fileName, mimeType (sniffed server-side, never trusted from the
client), sizeBytes, isImage, url, kind(file|image|voice|video),
durationMs?, waveform? (base64, ≤1024 bytes decoded), width?, height?.

Upload limits are clearance-differentiated and configured via env
(UPLOAD_MAX_MB_LEADERSHIP for levels 1–3, UPLOAD_MAX_MB_STAFF for 4–10);
clients read them from `GET /attachments/limit` — never hardcoded.

Two upload paths, both running the same content inspection (MIME sniffing +
executable rejection) before a file becomes servable:

- Single-shot `POST /attachments` (raw body, X-File-Name header, optional
  X-Attachment-Kind/-Duration-Ms/-Width/-Height metadata headers) — small
  files.
- Chunked: `POST /attachments/init` (declared size validated upfront) →
  `PUT /attachments/{id}/chunk?index=N` (idempotent by index; all chunks
  except the last are exactly chunkBytes) → `POST /attachments/{id}/complete`
  (assembles, inspects, creates the attachment, drops the session
  transactionally). `GET /attachments/{id}/upload-status` lists stored chunk
  indexes so interrupted clients resume instead of restarting. Sessions
  expire after UPLOAD_SESSION_TTL and are reaped hourly.

## Voice Messages (stage B)

A voice note is an Attachment with kind=voice, durationMs (mandatory,
1 ms – 10 min) and an optional waveform peak envelope, carried by an
attachment-only message. The server validates that the sniffed container is
one MediaRecorder produces (webm/ogg/mp4 family) and never trusts
client-declared MIME. The single-shot upload passes the waveform via the
X-Attachment-Waveform header (base64); the chunked init passes it in JSON.
No new endpoints or WS events — the existing message pipeline delivers voice
notes in both plaintext and E2EE chats. Attachment content encryption for
E2EE chats is a separate cross-cutting stage (docs/e2ee-design.md, этап 6)
covering all attachment kinds, voice included.

## Chat Media Aggregation (stage C)

`GET /chats/media?chatType=&chatId=&kind=media|files|links&cursor=&limit=`
serves the context panel tabs: media (image/video attachments), files
(plain file attachments; voice notes belong to neither tab), links (URLs
extracted server-side from plaintext message bodies only — E2EE ciphertext
is never scanned). Newest-first, (created_at, id) cursor pagination, items
carry messageId/senderId for jumping to the original. Access is the same
chat authorizer as the message list: non-members get a masked 404, deleted
messages are excluded. Addressing is by query parameters, consistent with
GET /messages (the /chats/{chatID} path wildcard owns that segment).

## Message Forwarding (stage D)

`POST /messages/forward { sourceMessageIds[], targetChatType, targetChatId }`
copies the actor's accessible PLAINTEXT messages into a target chat, order
preserved, each stamped with a forwarded-from author snapshot (id + display
name at forward time — never the source chat/message, so no cross-clearance
leak). Denormalized columns forwarded_from_message_id (audit only, not
exposed), forwarded_from_sender_id/name back the "Переслано от …" bubble.

Hierarchy rules (docs/spec/06, 07): the actor must be able to read every
source and post to the target (else masked 404 — inaccessible sources are
never revealed); the target's audience breadth must be ≤ every source's
(target min-clearance-level not weaker than the source), so content never
moves "up" to a broader audience (403). Each forward is audited
(message.forwarded).

E2EE: the server can't read ciphertext, so it rejects encrypted sources with
409; the client decrypts locally and re-sends via POST /messages with
forwardedFromSenderId/Name set (re-encrypting for an E2EE private target, or
sending the decrypted text when forwarding out to a non-E2EE target — an
explicit user action). Server-enforced hierarchy applies to the plaintext
path; for client-side E2EE forwards the server always enforces target access
but cannot police the source it cannot see.

## Text Formatting & Link Previews (stage E)

Message bodies support a markdown subset — **bold**, _italic_, ~~strike~~,
`inline code`, ```fenced code```, autolinks and @mentions. Rendered by
tokenizing into React nodes (shared/lib/richText), never via
dangerouslySetInnerHTML, so there is no HTML-injection surface; the stored
text is the raw source.

`POST /link-preview { url }` fetches OpenGraph metadata server-side. SSRF is
the core threat (docs/spec/06): scheme allowlist (http/https only); the
resolved IP is validated in the dialer Control hook right before connect, so
private/loopback/link-local (incl. 169.254.169.254)/CGNAT/unspecified are
refused and DNS-rebinding is defeated; redirects re-validated per hop, body
size and timeout capped; rate-limited (30/min); results cached in Redis
(failures negatively, shorter TTL). `GET /link-preview/image?url=` proxies
the preview image through the same guard so it loads same-origin under strict
CSP. In E2EE chats the server never sees message URLs, so the client requests
a preview only on an explicit user action; plaintext chats fetch
automatically.

## Mute & Notification Settings (stage G)

Per-user notification preferences. `PUT/DELETE /chats/{type}/{id}/mute`
mutes/unmutes a chat (untilSeconds = duration from now, 0/absent = forever);
`GET /settings/mutes` lists active mutes for the chat list.
`GET/PUT /settings/notifications` reads/updates settings: sound, preview
(text preview in notifications) and groupMode (all | mentions_only | none).

The notifications pipeline (internal/notifications.OnMessage) gates delivery
per recipient: a muted chat sends no push (the unread counter is still kept,
shown greyed); otherwise private chats always notify, and groups follow the
recipient's groupMode. @mentions still raise the in-app notification (bell)
unless muted. Message content stays out of push payloads (content-less, as
before); the preview flag is reserved for a future opt-in text preview.
Tables: chat_mutes, notification_settings (migration 000032).

## Chat Folders & Archive (UPD3 stage H)

Personal organizational metadata over chat references — never grants,
reveals, or widens access.

Folders: `GET /folders` (the actor's folders with items, ordered by
position), `POST /folders {name}` (max 50, name ≤ 64 chars),
`PATCH /folders/{id} {name}`, `DELETE /folders/{id}`,
`PUT /folders/order {folderIds}` (must list every folder exactly once),
`POST/DELETE /folders/{id}/items {chatType, chatId}`. Adding an item
verifies chat access; an inaccessible chat yields a masked 404 (its
existence is never confirmed). Removing never needs access — forgetting a
chat is always allowed.

Archive: `PUT/DELETE /chats/{chatType}/{chatID}/archive` (personal, like
mute; idempotent; masked 404 for inaccessible chats),
`GET /settings/archived` lists the actor's archived chat references.

Contract decision: `GET /chats` and `GET /groups` are intentionally
unchanged (additive contracts). Folder filtering and archive hiding happen
client-side against those access-filtered lists, so a folder item pointing
at a chat the user can no longer see simply never renders — a folder can
never leak a hidden chat. Tables: chat_folders, chat_folder_items,
chat_archives (migration 000033).

## Scheduled Messages (UPD3 stage I)

`POST /messages/schedule` freezes a send-body snapshot (same shape as
`POST /messages` plus `sendAt`, 5s..1y ahead, ≤100 pending per user);
`GET /messages/scheduled` lists the actor's rows (pending first);
`PATCH /messages/scheduled/{id}` edits content and/or `sendAt` while
pending; `DELETE` cancels and removes the snapshot entirely.

The worker replays due snapshots through the standard send pipeline
(`messages.SendTx`): the message insert and the row's `pending → sent`
flip share one transaction (`FOR UPDATE SKIP LOCKED`), so a crash or
concurrent replica can never double-send. Access is checked at scheduling
AND re-checked at send time with the sender's current role level; lost
access or a deactivated account cancels the row (content wiped) instead of
sending. Attachments are re-verified at send time: vanished files are
dropped, and a snapshot with nothing left is canceled. Push on delivery is
content-less and passes the stage-G mute/group-mode gate as usual.

E2EE ("path A", docs/security.md): the client encrypts at scheduling time
and submits ciphertext; the server stores and replays it without ever
reading it. The delivered message carries an additive `scheduledId` field
so the sender's client re-keys its locally cached plaintext
(`sched/<scheduledId>` → `msg/<messageId>`); the UI warns that key
rotation before `sendAt` can make the message undecryptable. Tables:
scheduled_messages + `messages.scheduled_message_id` (migration 000034).

## Disappearing Messages (UPD3 stage J)

`GET/PUT /chats/{chatType}/{chatID}/disappearing {ttlSeconds|null}` — the
chat's default timer (chat-wide, any member may change it; 5s..1y; audited
with TTL only, never content; masked 404 for inaccessible chats). New
messages then get `expiresAt = now + ttl`; a forward inherits the TARGET
chat's timer; a scheduled message sent into a disappearing chat expires at
`send_at + ttl` (the timer is applied at send time). The additive
`ttlSeconds` field of `POST /messages` and
`PUT /messages/{id}/expiry {ttlSeconds|null}` (sender-only) set a
per-message timer that overrides the chat default.

The reaper HARD-deletes expired rows — text, ciphertext and attachment
bytes leave the database (cascade), the search index is cleaned — and
publishes `message.deleted` with `expired: true`. Clients react by
dropping the bubble entirely (no tombstone) **and purging the locally
cached E2EE plaintext (`msg/<id>` in the IndexedDB keystore)**: without
that purge a "disappeared" message would silently survive client-side.
The same purge runs for ordinary deletions. `expiresAt` is metadata (like
`replyTo`): in E2EE chats the server sees when a message dies, never what
it said. This is a system deletion — users still cannot hard-delete each
other's messages. Tables: `messages.expires_at` + chat_disappear_settings
(migration 000035).
