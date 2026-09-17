-- Audit A-07: storage quotas and the posts-per-hour limit are checked on every
-- upload and every post, so their lookups must not scan whole tables.
-- (The existing idx_attachments_unlinked covers only unlinked rows.)
CREATE INDEX IF NOT EXISTS idx_attachments_uploaded_by ON attachments (uploaded_by);
CREATE INDEX IF NOT EXISTS idx_posts_author_recent ON posts (author_id, created_at DESC);
