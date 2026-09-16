ALTER TABLE groups DROP CONSTRAINT IF EXISTS groups_verified_by_with_at;
ALTER TABLE users DROP CONSTRAINT IF EXISTS users_verified_by_with_at;
ALTER TABLE groups DROP COLUMN IF EXISTS verified_by, DROP COLUMN IF EXISTS verified_at;
ALTER TABLE users DROP COLUMN IF EXISTS verified_by, DROP COLUMN IF EXISTS verified_at;
