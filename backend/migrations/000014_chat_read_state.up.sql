-- Per-user read position within a chat, used to compute unread counters.
-- A row is upserted whenever a user reads a conversation; unread messages
-- are those created after last_read_at that the user did not send.
CREATE TABLE chat_read_state (
    user_id             UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    chat_type           VARCHAR(16) NOT NULL CHECK (chat_type IN ('private', 'group')),
    chat_id             UUID NOT NULL,
    last_read_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    last_read_message_id UUID,
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT now(),

    PRIMARY KEY (user_id, chat_type, chat_id)
);

CREATE INDEX idx_chat_read_state_chat ON chat_read_state (chat_type, chat_id);

CREATE TRIGGER trg_chat_read_state_updated_at
    BEFORE UPDATE ON chat_read_state
    FOR EACH ROW
    EXECUTE FUNCTION set_updated_at();
