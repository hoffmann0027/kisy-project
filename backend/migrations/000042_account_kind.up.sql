-- Two kinds of account.
--
-- Until now every account arrived through a CEO invitation and therefore had a
-- clearance level 1..10, which is what the whole visibility model is built on
-- (roles.id IS the level — see 000002/000012). Registration is now also open
-- without an invitation, and such an account deliberately has NO level: it is
-- not "level 10", it is outside the hierarchy. Levels gate who may see which
-- group, who may be promoted and who may vote; an account that never passed
-- through an invitation takes no part in any of that.
--
-- Existing accounts default to 'invited': every one of them came through a
-- token, and none of them must lose its level.
ALTER TABLE users
    ADD COLUMN account_kind VARCHAR(16) NOT NULL DEFAULT 'invited'
        CHECK (account_kind IN ('basic', 'invited'));

ALTER TABLE users
    ALTER COLUMN role_id DROP NOT NULL;

-- The two columns describe one fact and must not be able to disagree. A basic
-- account with a level would silently pass clearance checks; an invited one
-- without a level could not be placed in the hierarchy at all.
ALTER TABLE users
    ADD CONSTRAINT users_account_kind_role CHECK (
        (account_kind = 'invited' AND role_id IS NOT NULL) OR
        (account_kind = 'basic' AND role_id IS NULL)
    );
