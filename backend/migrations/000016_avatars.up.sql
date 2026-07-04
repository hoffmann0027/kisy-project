-- Avatar images for users and groups, stored in the database so they survive
-- redeploys on platforms with ephemeral disks (e.g. Render free tier).
-- Images are validated and normalized (square, small) before insertion; the
-- users.avatar_url / groups.avatar_url columns point at GET /avatars/{type}/{id}.
CREATE TABLE avatars (
    owner_type   TEXT NOT NULL CHECK (owner_type IN ('user', 'group')),
    owner_id     UUID NOT NULL,
    content_type TEXT NOT NULL CHECK (content_type IN ('image/jpeg', 'image/png')),
    bytes        BYTEA NOT NULL,
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (owner_type, owner_id)
);
