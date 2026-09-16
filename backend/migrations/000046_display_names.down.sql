-- Back to free-form display names. Names chosen under the rule are kept; only
-- the rule, the key and the flag go.
ALTER TABLE users DROP CONSTRAINT IF EXISTS users_display_name_valid;
ALTER TABLE users DROP CONSTRAINT IF EXISTS users_display_name_valid_strict;
DROP INDEX IF EXISTS uq_users_display_name_key;
DROP INDEX IF EXISTS uq_users_display_name_key_all;
ALTER TABLE users DROP COLUMN IF EXISTS display_name_key;
ALTER TABLE users DROP COLUMN IF EXISTS display_name_needs_change;
DROP FUNCTION IF EXISTS kisy_display_name_valid(text);
DROP FUNCTION IF EXISTS kisy_display_name_key(text);
