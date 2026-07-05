-- Access level for a rating project (1 = CEO-only … 10 = everyone). A user of
-- clearance L sees a project when L <= min_level, mirroring group visibility.
ALTER TABLE rating_projects
    ADD COLUMN min_level INT NOT NULL DEFAULT 10 CHECK (min_level BETWEEN 1 AND 10);

CREATE INDEX idx_rating_projects_min_level ON rating_projects (min_level);
