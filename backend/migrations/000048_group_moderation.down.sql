-- Without the moderation tables a soft-deleted group would reappear as if
-- nothing had happened, so groups still waiting in the 30-day window are
-- removed first, with their messages, as the purge would have done.
DELETE FROM messages
WHERE chat_type = 'group'
  AND chat_id IN (SELECT id FROM groups WHERE deleted_at IS NOT NULL);
DELETE FROM groups WHERE deleted_at IS NOT NULL;

DROP TABLE IF EXISTS group_sanctions;
DROP INDEX IF EXISTS idx_groups_deleted_at;
ALTER TABLE groups DROP COLUMN IF EXISTS deleted_by, DROP COLUMN IF EXISTS deleted_at;
