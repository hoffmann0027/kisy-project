DROP INDEX IF EXISTS idx_rating_projects_status;
ALTER TABLE rating_projects DROP COLUMN IF EXISTS completed_at;
ALTER TABLE rating_projects DROP COLUMN IF EXISTS status;
