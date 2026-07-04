#!/usr/bin/env bash
# Restore the database from a gzip SQL backup produced by backup.sh.
# Usage: scripts/restore.sh [path-to-backup.sql.gz]  (defaults to newest)
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

set -a; [ -f .env ] && . ./.env; set +a
: "${POSTGRES_USER:?set in .env}" "${POSTGRES_DB:?set in .env}"

FILE="${1:-}"
if [ -z "$FILE" ]; then
  FILE="$(ls -1t "$ROOT"/backups/kisy-*.sql.gz 2>/dev/null | head -n1 || true)"
fi
[ -n "$FILE" ] && [ -f "$FILE" ] || { echo "No backup file found"; exit 1; }

echo "Restoring $FILE into $POSTGRES_DB (this overwrites current data)"
gzip -dc "$FILE" | docker compose exec -T postgres psql -U "$POSTGRES_USER" -d "$POSTGRES_DB"
echo "Restore complete."
