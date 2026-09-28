-- The comment migration 8 put on this table promised something that never
-- existed: "Not visible to recipients until scan_status = clean." There is no
-- scanner. The column is written 'clean' on insert and never read, so anyone
-- reading the schema was told a safety property the code does not have
-- (audit C-03).
--
-- Corrected here rather than by editing migration 8, so databases that already
-- applied it get the truth as well.
COMMENT ON TABLE attachments IS
    'Message attachments. NOTE: scan_status is written clean on insert and never read — there is no malware scanner (docs/security.md). Visibility is not gated on it.';

COMMENT ON COLUMN attachments.scan_status IS
    'Reserved for a future scanner (pending/clean/infected/failed). Always clean today.';
COMMENT ON COLUMN attachments.preview_path IS
    'Reserved for generated previews; nothing writes or reads it yet.';
COMMENT ON COLUMN attachments.scanned_at IS
    'Reserved for a future scanner; nothing writes or reads it yet.';
