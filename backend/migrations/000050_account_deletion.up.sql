-- Deleting your own account (audit E-01, Google Play "Account deletion").
--
-- The row itself cannot go: messages, posts and the audit journal point at it,
-- and dropping them would rewrite other people's conversations. So a deleted
-- account is anonymised — everything personal is removed, the row stays as an
-- anchor and shows as "Удалённый аккаунт".

ALTER TABLE users ADD COLUMN deleted_at TIMESTAMPTZ;

COMMENT ON TABLE users IS
    'An account is deactivated (is_active) or deleted by its owner (deleted_at: anonymised, never dropped).';

CREATE INDEX idx_users_deleted_at ON users (deleted_at) WHERE deleted_at IS NOT NULL;

-- Display names are unique (migration 46), but every deleted account carries
-- the same one. Exclude them from the uniqueness rule — they are not people
-- anyone can write to any more.
DROP INDEX uq_users_display_name_key;
CREATE UNIQUE INDEX uq_users_display_name_key ON users (display_name_key)
    WHERE NOT display_name_needs_change AND deleted_at IS NULL;

-- A deleted account's login is rewritten to "deleted_<hex>", which frees the
-- old one from the UNIQUE index. It must not become available again: the next
-- person to take it would inherit the conversations, mentions and links that
-- still name it.
CREATE TABLE retired_usernames (
    username   CITEXT PRIMARY KEY,
    retired_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

COMMENT ON TABLE retired_usernames IS
    'Logins of deleted accounts. Never reissued: an old link or mention must not resolve to someone new.';
