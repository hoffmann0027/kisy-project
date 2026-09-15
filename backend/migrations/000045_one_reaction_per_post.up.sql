-- One reaction per person per post.
--
-- Migration 43 carried the chat rule over — unique per (post, user, EMOJI) —
-- which let one reader stack every emoji on the same post. A chat reaction is
-- a quick reply to a message and several of them read fine; a reaction to a
-- post is a vote, and a vote is cast once. Picking another emoji now replaces
-- the first one instead of adding to it.
--
-- Rows that already break the new rule are resolved the way the new rule would
-- have resolved them: the latest choice wins. The id breaks a tie between two
-- rows inserted in the same instant, so exactly one row survives per pair.
DELETE FROM reactions r
USING reactions newer
WHERE r.post_id IS NOT NULL
  AND newer.post_id = r.post_id
  AND newer.user_id = r.user_id
  AND (newer.created_at, newer.id) > (r.created_at, r.id);

DROP INDEX uq_reactions_post;

CREATE UNIQUE INDEX uq_reactions_post_user ON reactions (post_id, user_id)
    WHERE post_id IS NOT NULL;

-- The new unique index leads with post_id, so it already serves the per-post
-- lookups idx_reactions_post existed for. Two indexes on one prefix would only
-- cost every write.
DROP INDEX idx_reactions_post;
