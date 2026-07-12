-- Stage K (UPD3): threads/discussions in group chats.

-- A thread is metadata over the same message stream: replies carry the
-- root's id and are excluded from the main feed. thread_root_id stays in
-- the clear even once groups become MLS-encrypted (E2EE stage 5) — like
-- reply_to, it reveals structure, never content. If a root is ever
-- hard-deleted (disappearing messages), its replies fall back to the main
-- feed rather than being destroyed with it.
ALTER TABLE messages
    ADD COLUMN thread_root_id UUID REFERENCES messages(id) ON DELETE SET NULL,
    -- Denormalized on the ROOT message so the feed shows "N replies · when"
    -- without counting per row.
    ADD COLUMN thread_reply_count INT NOT NULL DEFAULT 0,
    ADD COLUMN thread_last_reply_at TIMESTAMPTZ;

-- Thread page: replies of one root in chronological order.
CREATE INDEX idx_messages_thread ON messages (thread_root_id, created_at)
    WHERE thread_root_id IS NOT NULL;

-- Roots of a chat by latest activity (for "recent discussions" views).
CREATE INDEX idx_messages_thread_roots ON messages (chat_type, chat_id, thread_last_reply_at DESC)
    WHERE thread_last_reply_at IS NOT NULL;
