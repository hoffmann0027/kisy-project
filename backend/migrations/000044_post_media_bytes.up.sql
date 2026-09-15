-- Post media has to work on a deployment with no object storage.
--
-- Migration 43 gave post_media a storage_path and nothing else, which quietly
-- made the whole feature depend on BLOB_S3_* being configured. It is not
-- configured everywhere — and "you can write a post but not attach a photo,
-- depending on the deployment" is not a feature, it is a trap.
--
-- So the same shape attachments and avatars already use (migration 40): the
-- bytes live inline when there is nowhere better to put them, and in the
-- object store when there is.
ALTER TABLE post_media ADD COLUMN bytes BYTEA;
ALTER TABLE post_media ALTER COLUMN storage_path SET DEFAULT '';

-- Exactly one source of truth per row.
ALTER TABLE post_media ADD CONSTRAINT post_media_bytes_xor_path CHECK (
    (bytes IS NOT NULL AND storage_path = '') OR
    (bytes IS NULL AND storage_path <> '')
);
