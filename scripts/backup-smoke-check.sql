-- Restore-drill smoke check: run against a database that a backup was just
-- restored into. Any failed assertion RAISEs, so with psql -v ON_ERROR_STOP=1
-- the process exits non-zero and the drill (CI job) fails.
\set ON_ERROR_STOP on
DO $$
DECLARE
  migrations int;
  is_dirty   bool;
  t          text;
BEGIN
  -- The migration ledger must be present, non-empty and clean.
  IF to_regclass('public.schema_migrations') IS NULL THEN
    RAISE EXCEPTION 'schema_migrations table missing — restore did not complete';
  END IF;
  SELECT count(*) INTO migrations FROM schema_migrations;
  IF migrations < 1 THEN
    RAISE EXCEPTION 'schema_migrations is empty';
  END IF;
  SELECT bool_or(dirty) INTO is_dirty FROM schema_migrations;
  IF is_dirty THEN
    RAISE EXCEPTION 'schema_migrations is dirty';
  END IF;

  -- A representative set of core tables must be readable.
  FOREACH t IN ARRAY ARRAY['users', 'messages', 'groups', 'boards', 'call_logs'] LOOP
    IF to_regclass('public.' || t) IS NULL THEN
      RAISE EXCEPTION 'core table % missing after restore', t;
    END IF;
    EXECUTE format('SELECT 1 FROM %I LIMIT 1', t);
  END LOOP;

  RAISE NOTICE 'restore-drill smoke check passed (% migrations applied)', migrations;
END $$;
