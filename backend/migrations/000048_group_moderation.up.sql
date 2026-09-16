-- Moderation of groups and communities by the CEO: warnings, mutes, deletion.
--
-- Deletion by moderation is soft. For 30 days the group can be restored with
-- everything in it — posts, members, messages, board — and only then is it
-- removed for good, by the daily purge (internal/moderation). A founder's own
-- "delete group" is untouched and stays immediate.

ALTER TABLE groups
    ADD COLUMN deleted_at TIMESTAMPTZ,
    ADD COLUMN deleted_by UUID REFERENCES users(id) ON DELETE SET NULL;

-- The purge scans for groups past their 30 days; the admin screen lists the
-- deleted ones. Both look only at the few rows that are deleted.
CREATE INDEX idx_groups_deleted_at ON groups (deleted_at) WHERE deleted_at IS NOT NULL;

CREATE TABLE group_sanctions (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    group_id    UUID NOT NULL REFERENCES groups(id) ON DELETE CASCADE,
    kind        VARCHAR(8) NOT NULL CHECK (kind IN ('warn', 'mute', 'delete')),
    -- Shown to the group's founder and editors: a sanction without a reason is
    -- one nobody can learn from.
    reason      TEXT NOT NULL CHECK (char_length(btrim(reason)) BETWEEN 1 AND 1000),
    -- Accounts are deactivated, never deleted (docs/spec/01), so the issuer
    -- always exists.
    issued_by   UUID NOT NULL REFERENCES users(id),
    issued_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    -- Mutes only; NULL is indefinite.
    expires_at  TIMESTAMPTZ,
    -- A revoked sanction stays in the history and stops counting.
    revoked_at  TIMESTAMPTZ,
    revoked_by  UUID REFERENCES users(id),
    revoke_note TEXT,

    CONSTRAINT group_sanctions_expiry_only_for_mutes CHECK (kind = 'mute' OR expires_at IS NULL),
    CONSTRAINT group_sanctions_expiry_after_issue CHECK (expires_at IS NULL OR expires_at > issued_at),
    CONSTRAINT group_sanctions_revoked_by_with_at CHECK ((revoked_at IS NULL) = (revoked_by IS NULL))
);

COMMENT ON TABLE group_sanctions IS
    'CEO moderation of groups/communities. Active warn = not revoked; three active warns delete the group. Active mute = not revoked and not expired; a muted community is left out of GET /feed. See docs/spec/07-business-logic.md.';

CREATE INDEX idx_group_sanctions_group ON group_sanctions (group_id, issued_at DESC);

-- The feed asks "is this community muted right now?" for every page.
CREATE INDEX idx_group_sanctions_live_mutes ON group_sanctions (group_id, expires_at)
    WHERE kind = 'mute' AND revoked_at IS NULL;
