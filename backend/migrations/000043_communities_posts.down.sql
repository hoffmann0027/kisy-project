-- Reactions on posts cannot survive the loss of the posts table, so they go
-- first; message reactions are untouched.
DELETE FROM reactions WHERE post_id IS NOT NULL;

DROP INDEX IF EXISTS idx_reactions_post;
DROP INDEX IF EXISTS uq_reactions_post;
DROP INDEX IF EXISTS uq_reactions_message;

ALTER TABLE reactions DROP CONSTRAINT IF EXISTS reactions_exactly_one_target;
ALTER TABLE reactions DROP COLUMN IF EXISTS post_id;
ALTER TABLE reactions ALTER COLUMN message_id SET NOT NULL;
ALTER TABLE reactions
    ADD CONSTRAINT reactions_message_id_user_id_emoji_key UNIQUE (message_id, user_id, emoji);

DROP TABLE IF EXISTS feed_hidden_communities;
DROP TABLE IF EXISTS post_media;
DROP TABLE IF EXISTS posts;

DROP INDEX IF EXISTS idx_groups_public_community;
DROP INDEX IF EXISTS idx_groups_open;

-- A group without a threshold has no place in a schema where the threshold is
-- mandatory. It is given the weakest one rather than deleted: losing a group
-- and its messages to undo a schema change would be a far worse surprise.
UPDATE groups SET min_role_level = 10 WHERE min_role_level IS NULL;
ALTER TABLE groups ALTER COLUMN min_role_level SET NOT NULL;

ALTER TABLE groups DROP COLUMN IF EXISTS is_public;
ALTER TABLE groups DROP COLUMN IF EXISTS kind;
