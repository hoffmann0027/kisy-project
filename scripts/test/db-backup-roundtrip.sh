#!/usr/bin/env bash
# End-to-end test of the backup pipeline: scripts/db-backup.sh → scripts/db-restore.sh.
#
# It proves, against real Postgres containers and the real migrations:
#   1. a dump LARGER than a pipe buffer is kept (audit B-01: `grep -q` under
#      `pipefail` used to SIGPIPE gzip and delete a good dump);
#   2. with REQUIRE_ENCRYPTION=1 and no passphrase nothing is dumped at all
#      (audit A-08: an unencrypted dump must never be produced for upload);
#   3. the produced file is encrypted — no plaintext, not even gzip — and a
#      wrong passphrase cannot open it;
#   4. the dump restores into an empty database with every row and passes the
#      restore-drill smoke check.
#
#   bash scripts/test/db-backup-roundtrip.sh
#
# Needs docker, gpg and gzip. Cleans up its containers and network.
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
PG_IMAGE="${PG_IMAGE:-postgres:18-alpine}"
MIGRATE_IMAGE="${MIGRATE_IMAGE:-migrate/migrate:v4.19.1}"
SUFFIX="$$"
NET="kisy-backup-test-$SUFFIX"
SRC="kisy-backup-src-$SUFFIX"
DST="kisy-backup-dst-$SUFFIX"
WORK="$(mktemp -d)"
PASS="roundtrip-test-passphrase-$SUFFIX"
MARKER="ROUNDTRIP-PLAINTEXT-MARKER"
ROWS=5000

cleanup() {
  docker rm -f "$SRC" "$DST" >/dev/null 2>&1 || true
  docker network rm "$NET" >/dev/null 2>&1 || true
  rm -rf "$WORK"
}
trap cleanup EXIT

fail() { echo "FAIL: $*" >&2; exit 1; }
ok() { echo "ok - $*"; }

psql_in() { # container, sql
  docker exec -i "$1" psql -v ON_ERROR_STOP=1 -qtA -U postgres -d postgres -c "$2"
}

wait_ready() {
  for _ in $(seq 1 60); do
    if docker exec "$1" pg_isready -h 127.0.0.1 -U postgres >/dev/null 2>&1; then return 0; fi
    sleep 1
  done
  fail "postgres container $1 never became ready"
}

docker network create "$NET" >/dev/null
for c in "$SRC" "$DST"; do
  docker run -d --name "$c" --network "$NET" -e POSTGRES_PASSWORD=pw "$PG_IMAGE" >/dev/null
done
wait_ready "$SRC"
wait_ready "$DST"

# Real schema, so the smoke check sees exactly what production has.
MSYS_NO_PATHCONV=1 docker run --rm --network "$NET" -v "$ROOT/backend/migrations:/migrations:ro" \
  "$MIGRATE_IMAGE" -path /migrations \
  -database "postgres://postgres:pw@$SRC:5432/postgres?sslmode=disable" up >/dev/null

# Well past the 64 KiB pipe buffer once compressed data is decompressed.
psql_in "$SRC" "CREATE TABLE backup_roundtrip_filler (id int PRIMARY KEY, body text NOT NULL);
  INSERT INTO backup_roundtrip_filler
  SELECT g, '$MARKER ' || md5(g::text) || repeat('x', 200) FROM generate_series(1, $ROWS) g;" >/dev/null

SRC_URL="postgres://postgres:pw@$SRC:5432/postgres"
DST_URL="postgres://postgres:pw@$DST:5432/postgres"
export DOCKER_RUN_ARGS="--network $NET"
export PG_IMAGE

# --- 2. encryption required but no passphrase → refuse, produce nothing ---
set +e
REQUIRE_ENCRYPTION=1 BACKUP_GPG_PASSPHRASE="" DATABASE_URL="$SRC_URL" \
  bash "$ROOT/scripts/db-backup.sh" "$WORK/refused" >"$WORK/refused.log" 2>&1
status=$?
set -e
[ "$status" -ne 0 ] || fail "backup without a passphrase must fail when REQUIRE_ENCRYPTION=1"
if compgen -G "$WORK/refused/kisy-*" >/dev/null; then
  fail "a file was produced without a passphrase: $(ls "$WORK/refused")"
fi
# Refused up front, not after dumping plaintext to disk and deleting it.
grep -q "BACKUP_GPG_PASSPHRASE" "$WORK/refused.log" \
  || { cat "$WORK/refused.log" >&2; fail "refusal does not name the missing BACKUP_GPG_PASSPHRASE"; }
if grep -q "Dumping database" "$WORK/refused.log"; then
  fail "the database was dumped before the missing passphrase was noticed"
fi
ok "no passphrase + REQUIRE_ENCRYPTION=1 → refused, nothing written"

# --- 1 + 3. large encrypted dump is kept and is really encrypted ---
REQUIRE_ENCRYPTION=1 BACKUP_GPG_PASSPHRASE="$PASS" DATABASE_URL="$SRC_URL" \
  bash "$ROOT/scripts/db-backup.sh" "$WORK/out" >"$WORK/backup.log" 2>&1 \
  || { cat "$WORK/backup.log" >&2; fail "db-backup.sh failed on a large database"; }

mapfile -t files < <(ls "$WORK/out")
[ "${#files[@]}" -eq 1 ] || fail "expected exactly one file, got: ${files[*]:-none}"
FILE="$WORK/out/${files[0]}"
case "$FILE" in *.sql.gz.gpg) ;; *) fail "output is not an encrypted dump: $FILE" ;; esac
ok "large dump kept ($(wc -c <"$FILE") bytes): ${files[0]}"

if grep -a -q "$MARKER" "$FILE"; then fail "plaintext marker found in the encrypted file"; fi
if gzip -t "$FILE" 2>/dev/null; then fail "output is a readable gzip stream, not ciphertext"; fi
if gpg --batch --yes --quiet --pinentry-mode loopback --passphrase "wrong-$PASS" \
     --decrypt "$FILE" >/dev/null 2>&1; then
  fail "a wrong passphrase decrypted the dump"
fi
ok "file is ciphertext: no plaintext, not gzip, wrong passphrase rejected"

# --- 4. restore into an empty database and check everything came back ---
BACKUP_GPG_PASSPHRASE="$PASS" TARGET_DATABASE_URL="$DST_URL" \
  bash "$ROOT/scripts/db-restore.sh" "$FILE" >"$WORK/restore.log" 2>&1 \
  || { cat "$WORK/restore.log" >&2; fail "db-restore.sh failed"; }

got="$(psql_in "$DST" "SELECT count(*) FROM backup_roundtrip_filler")"
[ "$got" = "$ROWS" ] || fail "restored $got filler rows, expected $ROWS"
src_mig="$(psql_in "$SRC" "SELECT version FROM schema_migrations")"
dst_mig="$(psql_in "$DST" "SELECT version FROM schema_migrations")"
[ "$src_mig" = "$dst_mig" ] || fail "schema_migrations differs: src=$src_mig dst=$dst_mig"
docker exec -i "$DST" psql -v ON_ERROR_STOP=1 -q -U postgres -d postgres \
  <"$ROOT/scripts/backup-smoke-check.sql" >/dev/null \
  || fail "restore-drill smoke check failed on the restored database"
ok "restored $got rows, migration $dst_mig, smoke check passed"

echo "PASS: backup round-trip"
