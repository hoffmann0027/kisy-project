-- A sent scheduled message kept its content snapshot forever, after the
-- message it became was deleted or expired (audit A-29). New sends clear it
-- (scheduled.MarkSent); this clears what was sent before.
UPDATE scheduled_messages
SET text = NULL, ciphertext = NULL, attachment_ids = '{}'
WHERE status = 'sent';
