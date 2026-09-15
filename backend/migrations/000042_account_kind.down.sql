-- Rolling back means the hierarchy is mandatory again, so accounts that never
-- had a level have to be given one. They are placed at the bottom (10, the
-- level an invitation grants) rather than dropped: deleting people to undo a
-- schema change would be a far worse surprise than an unexpected level.
ALTER TABLE users
    DROP CONSTRAINT IF EXISTS users_account_kind_role;

UPDATE users SET role_id = 10 WHERE role_id IS NULL;

ALTER TABLE users
    ALTER COLUMN role_id SET NOT NULL;

ALTER TABLE users
    DROP COLUMN IF EXISTS account_kind;
