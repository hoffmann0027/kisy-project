-- Communities and their posts.
--
-- A community is a group with a different purpose: a group is a conversation
-- between its members, a community is a wall its owners publish to and its
-- members read. Same table, because everything else about them — membership,
-- roles, avatars, archiving — is identical, and splitting them would duplicate
-- all of it.

ALTER TABLE groups
    ADD COLUMN kind VARCHAR(16) NOT NULL DEFAULT 'group'
        CHECK (kind IN ('group', 'community'));

-- An open community's posts appear in the shared feed, and anyone may join it.
-- This is a different question from join_policy (migration 37), which says HOW
-- someone joins — by request or freely. A community can be closed to join yet
-- public to read, or open to join yet absent from the feed.
ALTER TABLE groups
    ADD COLUMN is_public BOOLEAN NOT NULL DEFAULT false;

-- No threshold at all: the group is open to every clearance, including
-- accounts that have none (users.account_kind = 'basic', migration 42). Until
-- now every group had to name a level, which left an account outside the
-- hierarchy unable to create one.
ALTER TABLE groups
    ALTER COLUMN min_role_level DROP NOT NULL;

-- The partial index lost its NULLs when the column became nullable; visibility
-- queries now ask for "no threshold OR threshold I clear", so both halves need
-- to be cheap.
CREATE INDEX idx_groups_open ON groups (id)
    WHERE is_archived = false AND min_role_level IS NULL;

CREATE INDEX idx_groups_public_community ON groups (created_at DESC)
    WHERE kind = 'community' AND is_public = true AND is_archived = false;

CREATE TABLE posts (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    community_id UUID NOT NULL REFERENCES groups(id) ON DELETE CASCADE,
    author_id    UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    text         TEXT NOT NULL DEFAULT '',
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    -- Soft delete: a post carries reactions and may be quoted, and the feed
    -- ranking reads history. Deleting the row outright would rewrite the past.
    deleted_at   TIMESTAMPTZ
);

COMMENT ON TABLE posts IS
    'Community posts. NOT end-to-end encrypted, unlike messages: a post is published to everyone who can see the community, and a feed the server cannot read cannot be ranked. See docs/e2ee-design.md.';

CREATE TRIGGER trg_posts_updated_at
    BEFORE UPDATE ON posts
    FOR EACH ROW
    EXECUTE FUNCTION set_updated_at();

-- The two shapes the feed asks for: one community's wall, newest first, and
-- the global feed across all of them.
CREATE INDEX idx_posts_community ON posts (community_id, created_at DESC)
    WHERE deleted_at IS NULL;
CREATE INDEX idx_posts_recent ON posts (created_at DESC)
    WHERE deleted_at IS NULL;

-- Media follows the shape attachments already use (migration 40): the bytes
-- live in object storage and the row keeps the key. There is no separate blob
-- table to point at — storage_path IS the reference.
CREATE TABLE post_media (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    post_id      UUID NOT NULL REFERENCES posts(id) ON DELETE CASCADE,
    kind         VARCHAR(16) NOT NULL CHECK (kind IN ('image', 'video', 'audio', 'file')),
    file_name    TEXT NOT NULL,
    mime_type    VARCHAR(128) NOT NULL,
    size_bytes   BIGINT NOT NULL CHECK (size_bytes > 0),
    storage_path TEXT NOT NULL,
    position     INT NOT NULL DEFAULT 0,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_post_media_post ON post_media (post_id, position);

-- --- Reactions become polymorphic -------------------------------------------
--
-- Posts reuse the reactions table rather than growing a parallel one: the rows,
-- the uniqueness rule and the counting are the same question. What is NOT the
-- same is who may react and who hears about it, and that stays in each module.
--
-- Two nullable columns with real foreign keys, not one bare target_id: a bare
-- id cannot cascade, and deleting a post would leave its reactions behind as
-- orphans pointing at nothing.
ALTER TABLE reactions ALTER COLUMN message_id DROP NOT NULL;

ALTER TABLE reactions
    ADD COLUMN post_id UUID REFERENCES posts(id) ON DELETE CASCADE;

ALTER TABLE reactions
    ADD CONSTRAINT reactions_exactly_one_target CHECK (
        (message_id IS NOT NULL AND post_id IS NULL) OR
        (message_id IS NULL AND post_id IS NOT NULL)
    );

-- The original UNIQUE (message_id, user_id, emoji) silently stops protecting
-- anything once message_id can be NULL: in SQL a NULL never equals a NULL, so
-- every post reaction would count as distinct and the same person could react
-- with the same emoji without limit. Replaced by two partial unique indexes,
-- one per target, each of which actually constrains its own rows.
ALTER TABLE reactions DROP CONSTRAINT reactions_message_id_user_id_emoji_key;

CREATE UNIQUE INDEX uq_reactions_message ON reactions (message_id, user_id, emoji)
    WHERE message_id IS NOT NULL;
CREATE UNIQUE INDEX uq_reactions_post ON reactions (post_id, user_id, emoji)
    WHERE post_id IS NOT NULL;

-- Counting distinct reactors per post is what the feed ranking runs on
-- (docs/spec/07-business-logic.md), so it gets its own index.
CREATE INDEX idx_reactions_post ON reactions (post_id)
    WHERE post_id IS NOT NULL;

-- Communities someone does not want to see in the shared feed.
CREATE TABLE feed_hidden_communities (
    user_id    UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    group_id   UUID NOT NULL REFERENCES groups(id) ON DELETE CASCADE,
    hidden_at  TIMESTAMPTZ NOT NULL DEFAULT now(),

    PRIMARY KEY (user_id, group_id)
);
