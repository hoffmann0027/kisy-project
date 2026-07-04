-- User feedback & suggestions. Any user may post; only the CEO may delete.
CREATE TABLE feedback (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    author_id  UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    body       TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_feedback_created_at ON feedback (created_at DESC, id DESC);
