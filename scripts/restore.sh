#!/usr/bin/env bash
# Restore the LOCAL docker-compose database from a backup made by backup.sh.
#
# This is the development path and nothing else: it talks to the `postgres`
# service of this compose project. Production runs on managed Postgres (Neon)
# and is restored with scripts/db-restore.sh — see docs/runbook.md, "Backup /
# restore". Pointing this script at production is impossible by construction,
# which is the intent (audit C-01).
#
# Usage: scripts/restore.sh [path-to-backup.sql.gz[.gpg]]  (defaults to newest)
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

set -a; [ -f .env ] && . ./.env; set +a
: "${POSTGRES_USER:?set in .env}" "${POSTGRES_DB:?set in .env}"

FILE="${1:-}"
if [ -z "$FILE" ]; then
  # Newest of either form: backup.sh writes .sql.gz, or .sql.gz.gpg when
  # BACKUP_GPG_RECIPIENT is set. Looking only for .sql.gz used to report "no
  # backup found" while an encrypted one sat right there.
  FILE="$(ls -1t "$ROOT"/backups/kisy-*.sql.gz "$ROOT"/backups/kisy-*.sql.gz.gpg 2>/dev/null | head -n1 || true)"
fi
[ -n "$FILE" ] && [ -f "$FILE" ] || { echo "No backup file found in $ROOT/backups" >&2; exit 1; }

echo "Restoring $FILE into $POSTGRES_DB (this overwrites current data)"

# ON_ERROR_STOP: without it psql skips failing statements and exits 0, so a
# dump applied over a non-empty database left a half-restored schema behind a
# cheerful "Restore complete." (audit C-01).
restore() {
  docker compose exec -T postgres psql -v ON_ERROR_STOP=1 --quiet \
    -U "$POSTGRES_USER" -d "$POSTGRES_DB"
}

case "$FILE" in
  *.gpg)
    command -v gpg >/dev/null || { echo "gpg is required to read $FILE" >&2; exit 1; }
    gpg --quiet --decrypt "$FILE" | gzip -dc | restore
    ;;
  *)
    gzip -dc "$FILE" | restore
    ;;
esac

echo "Restore complete: $FILE"
