-- Back to one reaction per (post, user, emoji). Every surviving row already
-- satisfies it — one per (post, user) is stricter — so nothing has to be
-- deleted on the way down.
CREATE INDEX idx_reactions_post ON reactions (post_id)
    WHERE post_id IS NOT NULL;

CREATE UNIQUE INDEX uq_reactions_post ON reactions (post_id, user_id, emoji)
    WHERE post_id IS NOT NULL;

DROP INDEX IF EXISTS uq_reactions_post_user;
