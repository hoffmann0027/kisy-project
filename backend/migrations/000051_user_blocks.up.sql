-- Blocking someone (audit E-02; Google Play requires it of any app where
-- strangers can write to you).
--
-- One-sided and silent: the person blocked is never told. What it changes is
-- in internal/blocks — new conversations, messages, the feed, people search
-- and calls.

CREATE TABLE user_blocks (
    blocker_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    blocked_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),

    PRIMARY KEY (blocker_id, blocked_id),
    CONSTRAINT user_blocks_not_self CHECK (blocker_id <> blocked_id)
);

-- The lookup that runs on every message: "did the other side block me?"
CREATE INDEX idx_user_blocks_blocked ON user_blocks (blocked_id);
