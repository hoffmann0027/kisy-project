-- Reporting content and people (audit E-02; Google Play requires a way to
-- report user-generated content and someone to act on it).
--
-- A report names what it is about, not what it says: the text of a private
-- message stays encrypted, and the queue shows that plainly.

CREATE TABLE reports (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    reporter_id  UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    target_kind  VARCHAR(16) NOT NULL CHECK (target_kind IN ('user', 'message', 'post', 'community')),
    target_id    UUID NOT NULL,
    -- Who authored what is being reported, captured now: the post may be
    -- deleted before anyone looks, and the count of reports against a person
    -- must survive that.
    target_owner UUID REFERENCES users(id) ON DELETE SET NULL,
    reason       VARCHAR(32) NOT NULL
                     CHECK (reason IN ('spam', 'abuse', 'fraud', 'illegal', 'other')),
    comment      TEXT,
    status       VARCHAR(16) NOT NULL DEFAULT 'open'
                     CHECK (status IN ('open', 'resolved', 'rejected')),
    resolved_by  UUID REFERENCES users(id),
    resolved_at  TIMESTAMPTZ,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),

    -- One report per person per thing: pressing twice is not a louder signal,
    -- and counting it as one is what makes "five different people" mean
    -- something.
    UNIQUE (reporter_id, target_kind, target_id)
);

-- The queue: open reports, newest first.
CREATE INDEX idx_reports_open ON reports (status, created_at DESC);

-- "How many people reported this post?" — the count behind auto-hiding.
CREATE INDEX idx_reports_target ON reports (target_kind, target_id) WHERE status = 'open';
