-- A project completes when all its tasks are done: its tasks are removed and
-- the project card moves to the "done" column. Track that with a status.
ALTER TABLE rating_projects
    ADD COLUMN status TEXT NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'done')),
    ADD COLUMN completed_at TIMESTAMPTZ;

CREATE INDEX idx_rating_projects_status ON rating_projects (status);
