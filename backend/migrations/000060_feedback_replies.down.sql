DROP INDEX IF EXISTS idx_feedback_author;
DROP INDEX IF EXISTS idx_feedback_unanswered;
ALTER TABLE feedback
    DROP CONSTRAINT IF EXISTS feedback_reply_shape,
    DROP COLUMN IF EXISTS replied_by,
    DROP COLUMN IF EXISTS replied_at,
    DROP COLUMN IF EXISTS reply;
