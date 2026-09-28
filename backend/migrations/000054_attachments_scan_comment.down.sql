-- Restore the wording migration 8 shipped with.
COMMENT ON TABLE attachments IS 'Not visible to recipients until scan_status = clean.';
COMMENT ON COLUMN attachments.scan_status IS NULL;
COMMENT ON COLUMN attachments.preview_path IS NULL;
COMMENT ON COLUMN attachments.scanned_at IS NULL;
