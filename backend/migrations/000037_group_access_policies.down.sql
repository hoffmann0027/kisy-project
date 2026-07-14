-- Demote any editors before restoring the narrower CHECK, so the down
-- migration cannot fail on existing rows.
UPDATE group_members SET role_in_group = 'moderator' WHERE role_in_group = 'editor';
ALTER TABLE group_members
    DROP CONSTRAINT IF EXISTS group_members_role_in_group_check;
ALTER TABLE group_members
    ADD CONSTRAINT group_members_role_in_group_check
        CHECK (role_in_group IN ('member', 'moderator', 'owner'));

ALTER TABLE groups
    DROP COLUMN IF EXISTS post_policy,
    DROP COLUMN IF EXISTS join_policy;
