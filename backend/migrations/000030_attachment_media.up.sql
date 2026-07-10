-- Stage A (update roadmap): attachment media metadata + chunked uploads.
--
-- `kind` and the media columns power voice messages (stage B) and the media
-- viewer (stage C). Chunked upload sessions hold partial data until
-- /complete assembles, inspects and promotes them into `attachments`;
-- stale sessions are reaped by a TTL worker.

ALTER TABLE attachments
    ADD COLUMN kind        VARCHAR(16) NOT NULL DEFAULT 'file'
        CHECK (kind IN ('file', 'image', 'voice', 'video')),
    ADD COLUMN duration_ms INTEGER CHECK (duration_ms IS NULL OR duration_ms >= 0),
    -- Compact peak envelope for voice bubbles (one byte per bar).
    ADD COLUMN waveform    BYTEA CHECK (waveform IS NULL OR octet_length(waveform) <= 1024),
    ADD COLUMN width       INTEGER CHECK (width  IS NULL OR width  > 0),
    ADD COLUMN height      INTEGER CHECK (height IS NULL OR height > 0);

-- Existing rows: classify images by their stored MIME.
UPDATE attachments SET kind = 'image' WHERE mime_type LIKE 'image/%';

CREATE TABLE attachment_upload_sessions (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    uploader       UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    file_name      TEXT NOT NULL,
    declared_bytes BIGINT NOT NULL CHECK (declared_bytes > 0),
    chunk_bytes    INTEGER NOT NULL CHECK (chunk_bytes > 0),
    kind           VARCHAR(16) NOT NULL DEFAULT 'file'
        CHECK (kind IN ('file', 'image', 'voice', 'video')),
    duration_ms    INTEGER CHECK (duration_ms IS NULL OR duration_ms >= 0),
    waveform       BYTEA CHECK (waveform IS NULL OR octet_length(waveform) <= 1024),
    width          INTEGER CHECK (width  IS NULL OR width  > 0),
    height         INTEGER CHECK (height IS NULL OR height > 0),
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    expires_at     TIMESTAMPTZ NOT NULL
);

CREATE INDEX idx_attachment_upload_sessions_expiry ON attachment_upload_sessions (expires_at);
CREATE INDEX idx_attachment_upload_sessions_uploader ON attachment_upload_sessions (uploader);

CREATE TABLE attachment_upload_chunks (
    session_id UUID NOT NULL REFERENCES attachment_upload_sessions(id) ON DELETE CASCADE,
    idx        INTEGER NOT NULL CHECK (idx >= 0),
    data       BYTEA NOT NULL,

    PRIMARY KEY (session_id, idx)
);
