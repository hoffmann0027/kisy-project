#!/usr/bin/env bash
# Makes the display-name rule unconditional, once nobody is left to rename.
#
#   DATABASE_URL='postgresql://…' scripts/finalize-display-names.sh
#
# Migration 46 introduced unique, letters-only display names without rewriting
# anyone's name: accounts whose name broke the rule or collided were flagged
# (users.display_name_needs_change) and are made to pick a new one at sign-in.
# Until they do, the unique index and the CHECK skip them.
#
# This script refuses to run while any account is still flagged. When none is:
#   1. builds a unique index over ALL rows, concurrently (no table lock);
#   2. adds the strict CHECK as NOT VALID, then VALIDATEs it (a light lock);
#   3. drops the lenient partial index and CHECK they replace.
# It is idempotent: re-running after success changes nothing.
#
# psql runs in a Postgres container, like the other database scripts.
set -euo pipefail

: "${DATABASE_URL:?set DATABASE_URL (the database to finalize)}"
PG_IMAGE="${PG_IMAGE:-postgres:18-alpine}"

psql_q() {
  docker run --rm -i "$PG_IMAGE" psql -v ON_ERROR_STOP=1 -qAt "$DATABASE_URL" "$@"
}

pending="$(psql_q -c "SELECT count(*) FROM users WHERE display_name_needs_change")"
if [ "$pending" != "0" ]; then
  echo "Not yet: $pending account(s) still have to choose a new display name." >&2
  psql_q -c "SELECT username || '  «' || display_name || '»' FROM users WHERE display_name_needs_change ORDER BY created_at" >&2
  exit 1
fi

echo "No flagged accounts. Building the unconditional unique index…"
# CONCURRENTLY cannot run inside a transaction block, hence its own call.
psql_q -c "CREATE UNIQUE INDEX CONCURRENTLY IF NOT EXISTS uq_users_display_name_key_all ON users (display_name_key)"

echo "Validating the strict rule…"
psql_q <<'SQL'
DO $$
BEGIN
  IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'users_display_name_valid_strict') THEN
    ALTER TABLE users ADD CONSTRAINT users_display_name_valid_strict
      CHECK (NOT display_name_needs_change AND kisy_display_name_valid(display_name)) NOT VALID;
  END IF;
END $$;
ALTER TABLE users VALIDATE CONSTRAINT users_display_name_valid_strict;
ALTER TABLE users DROP CONSTRAINT IF EXISTS users_display_name_valid;
DROP INDEX IF EXISTS uq_users_display_name_key;
SQL

echo "Done: display names are unique and valid for every account."
