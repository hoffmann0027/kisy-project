DROP INDEX idx_messages_thread_roots;
DROP INDEX idx_messages_thread;
ALTER TABLE messages
    DROP COLUMN thread_last_reply_at,
    DROP COLUMN thread_reply_count,
    DROP COLUMN thread_root_id;
