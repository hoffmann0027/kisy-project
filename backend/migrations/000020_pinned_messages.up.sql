-- Pinned messages: a chat participant may pin/unpin messages; pinned_at null
-- means not pinned.
ALTER TABLE messages ADD COLUMN pinned_at TIMESTAMPTZ;
ALTER TABLE messages ADD COLUMN pinned_by UUID REFERENCES users(id) ON DELETE SET NULL;

CREATE INDEX idx_messages_pinned ON messages (chat_type, chat_id) WHERE pinned_at IS NOT NULL;
