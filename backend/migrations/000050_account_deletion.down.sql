DROP TABLE IF EXISTS retired_usernames;

DROP INDEX IF EXISTS uq_users_display_name_key;
CREATE UNIQUE INDEX uq_users_display_name_key ON users (display_name_key)
    WHERE NOT display_name_needs_change;

DROP INDEX IF EXISTS idx_users_deleted_at;
ALTER TABLE users DROP COLUMN IF EXISTS deleted_at;

COMMENT ON TABLE users IS 'Accounts cannot be deleted, only deactivated via is_active.';
