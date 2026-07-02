CREATE TABLE attachments (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    message_id      UUID NOT NULL REFERENCES messages(id) ON DELETE CASCADE,
    file_name       TEXT NOT NULL,
    mime_type       VARCHAR(128) NOT NULL,
    size_bytes      BIGINT NOT NULL CHECK (size_bytes > 0),
    storage_path    TEXT NOT NULL,
    preview_path    TEXT,
    scan_status     VARCHAR(16) NOT NULL DEFAULT 'pending'
                        CHECK (scan_status IN ('pending', 'clean', 'infected', 'failed')),
    scanned_at      TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

COMMENT ON TABLE attachments IS 'Not visible to recipients until scan_status = clean.';

CREATE INDEX idx_attachments_message_id ON attachments (message_id);

CREATE TABLE reactions (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    message_id  UUID NOT NULL REFERENCES messages(id) ON DELETE CASCADE,
    user_id     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    emoji       VARCHAR(32) NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),

    UNIQUE (message_id, user_id, emoji)
);

CREATE INDEX idx_reactions_message_id ON reactions (message_id);

-- Mentions inside a message text, kept relational for notification fan-out.
CREATE TABLE message_mentions (
    message_id       UUID NOT NULL REFERENCES messages(id) ON DELETE CASCADE,
    mentioned_user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,

    PRIMARY KEY (message_id, mentioned_user_id)
);

CREATE INDEX idx_message_mentions_user_id ON message_mentions (mentioned_user_id);
