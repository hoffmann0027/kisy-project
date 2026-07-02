-- Groups are invisible to users whose role level is numerically greater
-- than min_role_level (lower clearance = higher level number).
CREATE TABLE groups (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name           VARCHAR(128) NOT NULL,
    description    TEXT,
    avatar_url     TEXT,
    min_role_level SMALLINT NOT NULL CHECK (min_role_level BETWEEN 1 AND 10),
    created_by     UUID NOT NULL REFERENCES users(id),
    is_archived    BOOLEAN NOT NULL DEFAULT false,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TRIGGER trg_groups_updated_at
    BEFORE UPDATE ON groups
    FOR EACH ROW
    EXECUTE FUNCTION set_updated_at();

CREATE INDEX idx_groups_min_role_level ON groups (min_role_level) WHERE is_archived = false;

CREATE TABLE group_members (
    group_id      UUID NOT NULL REFERENCES groups(id) ON DELETE CASCADE,
    user_id       UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role_in_group VARCHAR(16) NOT NULL DEFAULT 'member'
                      CHECK (role_in_group IN ('member', 'moderator', 'owner')),
    muted_until   TIMESTAMPTZ,
    joined_at     TIMESTAMPTZ NOT NULL DEFAULT now(),

    PRIMARY KEY (group_id, user_id)
);

CREATE INDEX idx_group_members_user_id ON group_members (user_id);
