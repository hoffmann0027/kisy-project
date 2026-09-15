-- Rows whose bytes live inline have nowhere to go once the column is gone, so
-- they go with it. Their posts survive, without the attachment.
DELETE FROM post_media WHERE bytes IS NOT NULL;

ALTER TABLE post_media DROP CONSTRAINT IF EXISTS post_media_bytes_xor_path;
ALTER TABLE post_media ALTER COLUMN storage_path DROP DEFAULT;
ALTER TABLE post_media DROP COLUMN IF EXISTS bytes;
