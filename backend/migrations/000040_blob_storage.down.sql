-- Reverting requires every avatar's bytes to be back in the database; rows
-- whose bytes only exist in object storage cannot satisfy the NOT NULL below.
-- Run the migrator in reverse (blobmigrate -direction=to-db) first.
DROP INDEX IF EXISTS idx_attachments_storage_path;

ALTER TABLE avatars DROP CONSTRAINT IF EXISTS avatars_bytes_xor_path;
ALTER TABLE avatars ALTER COLUMN bytes SET NOT NULL;
ALTER TABLE avatars DROP COLUMN storage_path;
