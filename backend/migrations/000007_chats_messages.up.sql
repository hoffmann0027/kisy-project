-- One-to-one conversations. A higher role level may initiate with a lower
-- one; once created either participant may reply until blocked (app layer).
CREATE TABLE private_chats (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_a_id   UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    user_b_id   UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    initiated_by UUID NOT NULL REFERENCES users(id),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT private_chats_distinct_users CHECK (user_a_id <> user_b_id)
);

-- Unordered-pair uniqueness: (A,B) and (B,A) are the same conversation.
CREATE UNIQUE INDEX idx_private_chats_pair
    ON private_chats (LEAST(user_a_id, user_b_id), GREATEST(user_a_id, user_b_id));

-- Messages belong to either a private_chats row or a groups row, selected
-- by chat_type; a single polymorphic FK is not expressible in SQL so
-- referential integrity for chat_id is enforced in the application layer.
CREATE TABLE messages (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    chat_type   VARCHAR(16) NOT NULL CHECK (chat_type IN ('private', 'group')),
    chat_id     UUID NOT NULL,
    sender_id   UUID NOT NULL REFERENCES users(id),
    text        TEXT,
    reply_to    UUID REFERENCES messages(id) ON DELETE SET NULL,
    is_deleted  BOOLEAN NOT NULL DEFAULT false,
    deleted_at  TIMESTAMPTZ,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT messages_deleted_consistency CHECK (
        (is_deleted = false AND deleted_at IS NULL) OR (is_deleted = true AND deleted_at IS NOT NULL)
    )
);

COMMENT ON TABLE messages IS 'Editing is disabled by product design; there is no edited_at column.';

CREATE INDEX idx_messages_chat_lookup ON messages (chat_type, chat_id, created_at DESC);
CREATE INDEX idx_messages_sender_id ON messages (sender_id);
CREATE INDEX idx_messages_reply_to ON messages (reply_to) WHERE reply_to IS NOT NULL;
