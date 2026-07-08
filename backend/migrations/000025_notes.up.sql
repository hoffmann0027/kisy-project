-- Personal notes: private scratchpad each user keeps for themselves. A note
-- carries free text and/or a single attached file (bytes stored in-row so they
-- survive redeploys on ephemeral disks). Only the owner can ever read or delete
-- their notes — enforced in the service, never merely hidden in the UI.
CREATE TABLE notes (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    text       TEXT,
    file_name  TEXT,
    file_type  TEXT,
    file_size  BIGINT NOT NULL DEFAULT 0,
    file_bytes BYTEA,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    -- A note must carry something: text, a file, or both.
    CONSTRAINT notes_nonempty CHECK (
        (text IS NOT NULL AND length(btrim(text)) > 0) OR file_bytes IS NOT NULL
    )
);

CREATE INDEX idx_notes_user ON notes (user_id, created_at DESC, id DESC);
