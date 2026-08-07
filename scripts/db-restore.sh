#!/usr/bin/env bash
# Restore a backup produced by db-backup.sh into a target Postgres
# (TARGET_DATABASE_URL) — e.g. a fresh Neon database during recovery.
#
#   TARGET_DATABASE_URL='postgresql://…neon.tech/neondb?sslmode=require' \
#     scripts/db-restore.sh backups/kisy-YYYY…Z.sql.gz
#
# Accepts .sql, .sql.gz or .sql.gz.gpg (the last needs BACKUP_GPG_PASSPHRASE).
# The target should be EMPTY — this loads the schema and data, it does not drop
# existing objects. psql runs in a Postgres 18 container (version-matched).
set -euo pipefail

: "${TARGET_DATABASE_URL:?set TARGET_DATABASE_URL (destination connection string)}"
FILE="${1:?usage: db-restore.sh <dump.sql[.gz][.gpg]>}"
[ -f "$FILE" ] || { echo "no such file: $FILE" >&2; exit 1; }
PG_IMAGE="${PG_IMAGE:-postgres:18-alpine}"

TMP="$(mktemp -d)"
trap 'rm -rf "$TMP"' EXIT
SQL="$TMP/dump.sql"

case "$FILE" in
  *.gpg)
    : "${BACKUP_GPG_PASSPHRASE:?encrypted backup needs BACKUP_GPG_PASSPHRASE}"
    gpg --batch --yes --decrypt --passphrase "$BACKUP_GPG_PASSPHRASE" "$FILE" | gzip -dc > "$SQL"
    ;;
  *.gz) gzip -dc "$FILE" > "$SQL" ;;
  *)    cp "$FILE" "$SQL" ;;
esac

echo "Restoring $FILE into the target database (loads schema + data)…"
docker run --rm -i "$PG_IMAGE" \
  psql -v ON_ERROR_STOP=1 "$TARGET_DATABASE_URL" < "$SQL"
echo "Restore complete."
