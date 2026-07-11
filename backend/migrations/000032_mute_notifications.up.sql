-- Stage G (update roadmap): per-user chat mutes + notification settings.

-- A mute suppresses push and notification.created for one chat, until
-- muted_until (NULL = muted forever). Unmuting deletes the row.
CREATE TABLE chat_mutes (
    user_id     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    chat_type   VARCHAR(16) NOT NULL CHECK (chat_type IN ('private', 'group')),
    chat_id     UUID NOT NULL,
    muted_until TIMESTAMPTZ,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),

    PRIMARY KEY (user_id, chat_type, chat_id)
);

CREATE INDEX idx_chat_mutes_lookup ON chat_mutes (user_id, chat_type, chat_id);

-- One settings row per user; absent rows fall back to defaults in code.
-- group_mode: how group chats notify — all messages, only @mentions, or
-- never. Private chats always notify (unless muted).
CREATE TABLE notification_settings (
    user_id    UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    sound      BOOLEAN NOT NULL DEFAULT true,
    preview    BOOLEAN NOT NULL DEFAULT true,
    group_mode VARCHAR(16) NOT NULL DEFAULT 'all'
        CHECK (group_mode IN ('all', 'mentions_only', 'none')),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
