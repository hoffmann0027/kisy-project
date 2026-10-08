-- Announcements: notifications written by people, not raised by the system.
-- Levels 1-3 send them to everyone, to basic accounts, to chosen levels or to
-- one person — always downwards or sideways in the hierarchy
-- (internal/announcements). Each recipient gets an ordinary notifications row
-- pointing back here, so revoking one removes it from every list at once.
CREATE TABLE announcements (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    author_id       UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    audience        VARCHAR(16) NOT NULL CHECK (audience IN ('all', 'basic', 'levels', 'user')),
    -- audience = 'levels': which levels; 'user': who.
    levels          SMALLINT[],
    target_user_id  UUID REFERENCES users(id) ON DELETE CASCADE,
    title           VARCHAR(100) NOT NULL CHECK (length(btrim(title)) > 0),
    body            VARCHAR(1000) NOT NULL CHECK (length(btrim(body)) > 0),
    recipient_count INTEGER NOT NULL DEFAULT 0,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    revoked_at      TIMESTAMPTZ,
    revoked_by      UUID REFERENCES users(id) ON DELETE SET NULL,

    CONSTRAINT announcements_audience_shape CHECK (
        (audience = 'levels' AND levels IS NOT NULL AND cardinality(levels) > 0 AND target_user_id IS NULL) OR
        (audience = 'user' AND target_user_id IS NOT NULL AND levels IS NULL) OR
        (audience IN ('all', 'basic') AND levels IS NULL AND target_user_id IS NULL)
    )
);

-- The author's history, newest first, and the daily quota count.
CREATE INDEX idx_announcements_author ON announcements (author_id, created_at DESC);
CREATE INDEX idx_announcements_created ON announcements (created_at DESC);

ALTER TABLE notifications
    ADD COLUMN announcement_id UUID REFERENCES announcements(id) ON DELETE CASCADE;

CREATE INDEX idx_notifications_announcement ON notifications (announcement_id)
    WHERE announcement_id IS NOT NULL;
