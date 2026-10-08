DROP INDEX IF EXISTS idx_notifications_announcement;
ALTER TABLE notifications DROP COLUMN IF EXISTS announcement_id;
DROP TABLE IF EXISTS announcements;
