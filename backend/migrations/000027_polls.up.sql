-- Voting ("Голосование"): the CEO creates a poll with a description and up to
-- ten options; every member casts one vote (changeable while the poll is
-- open). Votes are public — each option shows the avatars of who chose it — so
-- there is no anonymity to protect here. When the CEO closes a poll, the tally
-- is written to the audit log.
CREATE TABLE polls (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    question   TEXT NOT NULL,
    status     TEXT NOT NULL DEFAULT 'open' CHECK (status IN ('open', 'closed')),
    created_by UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    closed_at  TIMESTAMPTZ
);

CREATE TABLE poll_options (
    id       UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    poll_id  UUID NOT NULL REFERENCES polls(id) ON DELETE CASCADE,
    position INT  NOT NULL,
    body     TEXT NOT NULL
);
CREATE INDEX idx_poll_options_poll ON poll_options (poll_id, position);

CREATE TABLE poll_votes (
    poll_id    UUID NOT NULL REFERENCES polls(id) ON DELETE CASCADE,
    option_id  UUID NOT NULL REFERENCES poll_options(id) ON DELETE CASCADE,
    user_id    UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    -- One vote per user per poll (changing a vote updates option_id).
    PRIMARY KEY (poll_id, user_id)
);
CREATE INDEX idx_poll_votes_option ON poll_votes (option_id);

CREATE INDEX idx_polls_created_at ON polls (created_at DESC, id DESC);
