-- Feedback becomes a private line to leadership (October 2026): the author
-- sees only their own, levels 1-3 see what nobody has answered yet and reply
-- once. An answered entry leaves their inbox and shows the reply to its author.
ALTER TABLE feedback
    ADD COLUMN reply      TEXT,
    ADD COLUMN replied_at TIMESTAMPTZ,
    ADD COLUMN replied_by UUID REFERENCES users(id) ON DELETE SET NULL,
    ADD CONSTRAINT feedback_reply_shape CHECK (
        (reply IS NULL AND replied_at IS NULL) OR
        (reply IS NOT NULL AND replied_at IS NOT NULL AND length(btrim(reply)) > 0)
    );

-- The inbox: unanswered, newest first.
CREATE INDEX idx_feedback_unanswered ON feedback (created_at DESC, id DESC) WHERE replied_at IS NULL;
-- An author's own list, and the once-a-day check.
CREATE INDEX idx_feedback_author ON feedback (author_id, created_at DESC);
