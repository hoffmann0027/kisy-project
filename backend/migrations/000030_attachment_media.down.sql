DROP TABLE attachment_upload_chunks;
DROP TABLE attachment_upload_sessions;

ALTER TABLE attachments
    DROP COLUMN kind,
    DROP COLUMN duration_ms,
    DROP COLUMN waveform,
    DROP COLUMN width,
    DROP COLUMN height;
