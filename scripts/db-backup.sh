#!/usr/bin/env bash
# Back up a Postgres database (DATABASE_URL) to a gzip'd plain-SQL file.
#
# pg_dump runs inside a Postgres 18 container so the client version always
# matches the Neon server (18) and no local install is needed — works the same
# on a laptop and in CI. Optionally symmetric-encrypts the result with GPG.
#
#   DATABASE_URL='postgresql://…neon.tech/neondb?sslmode=require' \
#     scripts/db-backup.sh [out-dir]
#
# Set BACKUP_GPG_PASSPHRASE to encrypt (AES-256): the dump is then streamed
# straight into gpg and plaintext never touches the disk. REQUIRE_ENCRYPTION=1
# refuses to run at all without a passphrase — CI sets it, because a backup that
# ends up anywhere shared must never be readable. PG_IMAGE overrides the
# Postgres image tag (keep its major version >= the server's).
set -euo pipefail
# DOCKER_RUN_ARGS: extra `docker run` flags for the client container (e.g.
# "--network kisy-test" so a test can reach a database in another container).

: "${DATABASE_URL:?set DATABASE_URL (Neon direct connection string)}"
# A bare dbname or truncated value makes pg_dump silently fall back to a local
# unix socket ("connection to server on socket … No such file"); catch that
# early with a clear message instead.
case "$DATABASE_URL" in
  postgres://* | postgresql://*) ;;
  *)
    echo "ERROR: DATABASE_URL must be a full postgresql:// connection string" >&2
    echo "       (with host), e.g. postgresql://user:pass@host/db?sslmode=require" >&2
    exit 1
    ;;
esac
if [ "${REQUIRE_ENCRYPTION:-}" = "1" ] && [ -z "${BACKUP_GPG_PASSPHRASE:-}" ]; then
  echo "ERROR: REQUIRE_ENCRYPTION=1 but BACKUP_GPG_PASSPHRASE is empty — refusing to dump" >&2
  exit 1
fi
OUT_DIR="${1:-backups}"
PG_IMAGE="${PG_IMAGE:-postgres:18-alpine}"

mkdir -p "$OUT_DIR"
STAMP="$(date -u +%Y%m%dT%H%M%SZ)"
FILE="$OUT_DIR/kisy-$STAMP.sql.gz"
ENCRYPT=""
if [ -n "${BACKUP_GPG_PASSPHRASE:-}" ]; then
  ENCRYPT=1
  FILE="$FILE.gpg"
fi

# The passphrase goes to gpg on a file descriptor, never on the command line
# (where any process listing would show it).
encrypt() {
  gpg --batch --yes --quiet --pinentry-mode loopback --symmetric --cipher-algo AES256 \
    --passphrase-fd 3 3<<<"$BACKUP_GPG_PASSPHRASE"
}
decrypt() {
  gpg --batch --quiet --pinentry-mode loopback --decrypt \
    --passphrase-fd 3 "$1" 3<<<"$BACKUP_GPG_PASSPHRASE"
}

echo "Dumping database -> $FILE"
# --no-owner/--no-acl drop role and grant statements tied to the source
# (e.g. Neon's neondb_owner), so the dump restores cleanly into any fresh DB.
# shellcheck disable=SC2086 # DOCKER_RUN_ARGS is a word list on purpose
if [ -n "$ENCRYPT" ]; then
  docker run --rm ${DOCKER_RUN_ARGS:-} "$PG_IMAGE" \
    pg_dump --no-owner --no-acl --format=plain "$DATABASE_URL" \
    | gzip -9 | encrypt > "$FILE"
else
  docker run --rm ${DOCKER_RUN_ARGS:-} "$PG_IMAGE" \
    pg_dump --no-owner --no-acl --format=plain "$DATABASE_URL" \
    | gzip -9 > "$FILE"
fi

# Guard against a truncated/empty dump masquerading as success. grep -c reads
# the whole stream: grep -q would exit on the first match, the upstream gzip
# would die of SIGPIPE, and under pipefail a GOOD dump would be reported as
# empty and deleted — which is exactly how backups silently stopped (audit B-01).
if [ -n "$ENCRYPT" ]; then
  tables="$(decrypt "$FILE" | gzip -dc | grep -c "^CREATE TABLE" || true)"
else
  tables="$(gzip -dc "$FILE" | grep -c "^CREATE TABLE" || true)"
fi
if [ "${tables:-0}" -lt 1 ]; then
  echo "ERROR: dump contains no tables — aborting" >&2
  rm -f "$FILE"
  exit 1
fi

echo "Backup complete: $FILE ($(du -h "$FILE" | cut -f1), $tables tables${ENCRYPT:+, encrypted})"
