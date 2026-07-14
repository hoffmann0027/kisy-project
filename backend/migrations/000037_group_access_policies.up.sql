-- Group access settings (Stage N): two independent axes.
--   join_policy — how cleared users join: 'open' (self-join) | 'request' (apply).
--   post_policy — who may post:          'all' (any member) | 'editors' (owner/editor/moderator/CEO).
-- Defaults preserve today's private behaviour: existing groups become
-- request-to-join with open posting (no existing group is weakened).
ALTER TABLE groups
    ADD COLUMN join_policy VARCHAR(16) NOT NULL DEFAULT 'request'
        CHECK (join_policy IN ('open', 'request')),
    ADD COLUMN post_policy VARCHAR(16) NOT NULL DEFAULT 'all'
        CHECK (post_policy IN ('all', 'editors'));

-- Stage L's `editor` role never landed here (role_in_group was
-- member|moderator|owner); add it so the "editor tier" for posting and
-- request approval is reachable.
ALTER TABLE group_members
    DROP CONSTRAINT IF EXISTS group_members_role_in_group_check;
ALTER TABLE group_members
    ADD CONSTRAINT group_members_role_in_group_check
        CHECK (role_in_group IN ('member', 'moderator', 'editor', 'owner'));
