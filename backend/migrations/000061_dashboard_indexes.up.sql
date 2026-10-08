-- The admin overview counts the last 24 hours of messages and sessions. The
-- existing message indexes all lead with the chat, so without these the count
-- reads the whole table.
CREATE INDEX IF NOT EXISTS idx_messages_created_at ON messages (created_at);
CREATE INDEX IF NOT EXISTS idx_sessions_last_used_at ON sessions (last_used_at);
