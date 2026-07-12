DROP TABLE chat_disappear_settings;
DROP INDEX idx_messages_expiring;
ALTER TABLE messages DROP COLUMN expires_at;
