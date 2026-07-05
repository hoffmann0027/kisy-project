DROP INDEX IF EXISTS idx_rating_projects_min_level;
ALTER TABLE rating_projects DROP COLUMN IF EXISTS min_level;
