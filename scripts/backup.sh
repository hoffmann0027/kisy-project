#!/usr/bin/env bash
# Backup of the LOCAL docker-compose database. Dumps the running `postgres`
# service to backups/ as a gzip-compressed SQL file; set BACKUP_GPG_RECIPIENT
# to encrypt it with GPG.
#
# NOT the production path. Production runs on managed Postgres (Neon), which
# this script cannot reach: it goes through `docker compose exec`. The
# production backup is the "DB backup" workflow and scripts/db-backup.sh —
# see docs/runbook.md, "Backup / restore" (audit C-01).
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

# Load DB credentials from .env.
set -a; [ -f .env ] && . ./.env; set +a
: "${POSTGRES_USER:?set in .env}" "${POSTGRES_DB:?set in .env}"

OUT_DIR="$ROOT/backups"
mkdir -p "$OUT_DIR"
STAMP="$(date -u +%Y%m%dT%H%M%SZ)"
FILE="$OUT_DIR/kisy-$STAMP.sql.gz"

echo "Dumping $POSTGRES_DB -> $FILE"
docker compose exec -T postgres pg_dump -U "$POSTGRES_USER" -d "$POSTGRES_DB" \
  | gzip -9 > "$FILE"

if [ -n "${BACKUP_GPG_RECIPIENT:-}" ]; then
  gpg --yes --encrypt --recipient "$BACKUP_GPG_RECIPIENT" "$FILE"
  rm -f "$FILE"
  FILE="$FILE.gpg"
  echo "Encrypted -> $FILE"
fi

# Retention: keep the newest 14 backups.
ls -1t "$OUT_DIR"/kisy-*.sql.gz* 2>/dev/null | tail -n +15 | xargs -r rm -f

echo "Backup complete: $FILE"
