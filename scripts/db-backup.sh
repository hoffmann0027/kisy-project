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
# Set BACKUP_GPG_PASSPHRASE to also encrypt (AES-256). PG_IMAGE overrides the
# Postgres image tag (keep its major version >= the server's).
set -euo pipefail

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
OUT_DIR="${1:-backups}"
PG_IMAGE="${PG_IMAGE:-postgres:18-alpine}"

mkdir -p "$OUT_DIR"
STAMP="$(date -u +%Y%m%dT%H%M%SZ)"
FILE="$OUT_DIR/kisy-$STAMP.sql.gz"

echo "Dumping database -> $FILE"
# --no-owner/--no-acl drop role and grant statements tied to the source
# (e.g. Neon's neondb_owner), so the dump restores cleanly into any fresh DB.
docker run --rm "$PG_IMAGE" \
  pg_dump --no-owner --no-acl --format=plain "$DATABASE_URL" \
  | gzip -9 > "$FILE"

# Guard against a truncated/empty dump masquerading as success.
if ! gzip -dc "$FILE" | grep -q "CREATE TABLE"; then
  echo "ERROR: dump contains no tables — aborting" >&2
  rm -f "$FILE"
  exit 1
fi

if [ -n "${BACKUP_GPG_PASSPHRASE:-}" ]; then
  gpg --batch --yes --symmetric --cipher-algo AES256 \
    --passphrase "$BACKUP_GPG_PASSPHRASE" -o "$FILE.gpg" "$FILE"
  rm -f "$FILE"
  FILE="$FILE.gpg"
  echo "Encrypted -> $FILE"
fi

echo "Backup complete: $FILE ($(du -h "$FILE" | cut -f1))"
