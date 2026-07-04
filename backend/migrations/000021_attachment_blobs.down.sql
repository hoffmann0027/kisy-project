DROP INDEX IF EXISTS idx_attachments_unlinked;
ALTER TABLE attachments ALTER COLUMN storage_path DROP DEFAULT;
DELETE FROM attachments WHERE message_id IS NULL;
ALTER TABLE attachments ALTER COLUMN message_id SET NOT NULL;
ALTER TABLE attachments DROP COLUMN IF EXISTS uploaded_by;
ALTER TABLE attachments DROP COLUMN IF EXISTS data;
