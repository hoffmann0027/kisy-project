-- Stage J (UPD3): disappearing messages.

-- When a message self-destructs. NULL = never. Metadata only (like
-- reply_to): in E2EE chats the server sees the timer but never the content.
ALTER TABLE messages ADD COLUMN expires_at TIMESTAMPTZ;

-- The reaper scans only rows that actually expire.
CREATE INDEX idx_messages_expiring ON messages (expires_at)
    WHERE expires_at IS NOT NULL;

-- Default timer for NEW messages of a chat. A chat-wide property (as in
-- Signal/Telegram), not per-user: disappearing is a mode of the
-- conversation, visible to every participant. Enabling/changing is audited.
CREATE TABLE chat_disappear_settings (
    chat_type   VARCHAR(16) NOT NULL CHECK (chat_type IN ('private', 'group')),
    chat_id     UUID NOT NULL,
    ttl_seconds BIGINT NOT NULL CHECK (ttl_seconds > 0),
    set_by      UUID REFERENCES users(id) ON DELETE SET NULL,
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now(),

    PRIMARY KEY (chat_type, chat_id)
);
