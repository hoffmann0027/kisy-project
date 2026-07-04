-- Allow editing a message: records when it was last edited (null = never).
ALTER TABLE messages ADD COLUMN edited_at TIMESTAMPTZ;
