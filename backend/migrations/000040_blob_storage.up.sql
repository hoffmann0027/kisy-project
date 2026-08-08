-- Object storage for uploaded bytes (O5). Files may now live in an
-- S3-compatible bucket instead of the database: the row keeps the object key
-- and the BYTEA column stays NULL. Both shapes are readable at once, so the
-- migration of existing rows can run gradually with no downtime.

-- Avatars: add the key column and let the bytes go empty once moved.
ALTER TABLE avatars ADD COLUMN storage_path TEXT NOT NULL DEFAULT '';
ALTER TABLE avatars ALTER COLUMN bytes DROP NOT NULL;

-- Exactly one source of truth per row: inline bytes or an object key.
ALTER TABLE avatars ADD CONSTRAINT avatars_bytes_xor_path CHECK (
    (bytes IS NOT NULL AND storage_path = '') OR
    (bytes IS NULL AND storage_path <> '')
);

-- Attachments already carry storage_path (unused until now); data is already
-- nullable. Index the key so the "is this object still referenced?" check the
-- disappearing-message reaper runs before purging bytes stays cheap.
CREATE INDEX idx_attachments_storage_path ON attachments (storage_path)
    WHERE storage_path <> '';
