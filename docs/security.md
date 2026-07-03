# KISY Security Model

This document records the threat model (STRIDE) and the controls that
implement `docs/spec/06-security.md`. It is the reference for the
security-hardening pass and for future audits.

## Principles

Defense in depth, least privilege, secure-by-default, zero-trust between
tiers. Every privileged action is audited; internal errors are never
surfaced to clients.

## STRIDE threat model

### Spoofing (impersonation)
- **Threats:** credential theft, session hijacking, forged identity.
- **Mitigations:** Argon2id password hashing (`internal/auth/password`);
  JWT access tokens signed HS256 with a ≥32-char secret; opaque refresh
  tokens stored only as SHA-256 digests; HTTPOnly + `SameSite=Strict`
  cookies; per-request session validation (logout/password-change take
  effect immediately); account lockout after 5 failures / 15 min; per-IP
  rate limiting (Redis + Nginx). Timing-equalized login avoids user
  enumeration.

### Tampering (unauthorized modification)
- **Threats:** request/response tampering, SQL injection, message forgery.
- **Mitigations:** parameterized queries everywhere (pgx, no string
  concatenation); strict JSON decoding with unknown-field rejection and a
  1 MiB body cap (`pkg/httpjson`); refresh-token rotation with reuse
  detection (a replayed token revokes the session); TLS 1.3 in transit;
  append-only `audit_logs` (UPDATE/DELETE blocked by a DB trigger).

### Repudiation (denying an action)
- **Threats:** a user or admin denies performing an action.
- **Mitigations:** immutable audit log capturing actor, action, target,
  hashed IP, session id, request id and timestamp for every privileged
  operation (login, logout, register, invite create/use, role change,
  password reset, activate/deactivate, message deletion, group creation,
  refresh-reuse security events). Readable only by the CEO via `GET
  /admin/audit`.

### Information Disclosure
- **Threats:** leaking data or resource existence across clearance levels.
- **Mitigations:** clearance model (`internal/access`) — groups above a
  user's clearance are invisible and return **404, not 403**, so their
  existence cannot be probed; message/chat access failures collapse to
  not-found; the user directory only returns same-or-lower-clearance
  accounts; passwords/refresh tokens never leave the backend; IP addresses
  are stored salted-hashed; internal errors return a generic message with
  a request id, never a stack trace.

### Denial of Service
- **Threats:** brute force, request floods, slowloris, oversized payloads.
- **Mitigations:** Nginx `limit_req` zones (auth 5 r/s, api 30 r/s per IP)
  plus Redis per-IP limits in the backend; account lockout; request
  header/body timeouts and `client_max_body_size`; server read-header
  timeout; WebSocket read limits and slow-consumer frame dropping;
  graceful shutdown.

### Elevation of Privilege
- **Threats:** a lower-clearance user gaining higher access.
- **Mitigations:** RBAC middleware (`RequireAuth` + `RequireClearance`),
  1 = CEO; admin surface gated at clearance 1; a CEO cannot demote or
  deactivate themselves (no self-lockout); the private-chat initiation
  rule forbids reaching upward; role level is embedded in the validated
  access token and re-checked against the live session on every request.

## Application-security controls

| Attack               | Control                                                             |
|----------------------|---------------------------------------------------------------------|
| XSS                  | Strict CSP (`script-src 'self'`, no inline scripts); React escaping  |
| CSRF                 | `SameSite=Strict` cookies + Origin/Referer verification middleware  |
| Clickjacking         | `X-Frame-Options: DENY` + CSP `frame-ancestors 'none'`              |
| SQL injection        | Parameterized pgx queries only                                      |
| MIME sniffing        | `X-Content-Type-Options: nosniff`                                    |
| Open redirect        | No user-controlled redirects; `form-action 'self'`, `base-uri 'self'`|
| Path traversal / IDOR| UUID keys; per-resource ownership/clearance checks; not-found masking|
| Info leakage in errors| Generic client errors, structured server-side logs                 |
| Transport downgrade  | HSTS (2 y, preload) + TLS 1.3-only + HTTP→HTTPS 308 redirect        |

## Transport & headers

- **TLS:** production terminates TLS 1.3 only at the Nginx edge
  (`deploy/nginx/nginx.tls.conf`), PFS-inherent cipher suites, HSTS with
  preload, `http2`. Dev certs via `scripts/gen-dev-certs.sh`; enable with
  `docker compose -f docker-compose.yml -f docker-compose.prod.yml up`.
- **Headers:** CSP, `X-Content-Type-Options`, `X-Frame-Options`,
  `Referrer-Policy`, `Permissions-Policy`, `Cross-Origin-*` set by the
  backend (`internal/platform/security`) on API responses and by Nginx on
  static responses; `server_tokens off` hides the version.

## Secrets

All secrets (DB/Redis passwords, JWT keys, IP-hash salt, CEO bootstrap
credentials) come from environment variables / `.env`, never hardcoded.
`.env` and TLS private keys are git-ignored.

## Dependency hygiene

`govulncheck ./...` (Go) and `npm audit` (frontend) run clean (0
vulnerabilities) as of the stage-6 pass and are wired into CI.

## Deferred to later stages

- Malware/virus scanning of uploaded files (Files stage; the `attachments`
  table already gates visibility on `scan_status = clean`).
- Encryption at rest for selected sensitive columns.
- CAPTCHA after repeated failures (lockout + rate limiting cover the
  baseline).
- Automated backup verification jobs.
