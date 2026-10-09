-- Who answers for a project: its members record its income and expenses
-- (every ledger row already names its author), and the project's row shows
-- them as its team. The project's creator (levels 1–4) and the CEO change the
-- list; the people on it change over time.
CREATE TABLE rating_project_members (
    project_id UUID NOT NULL REFERENCES rating_projects(id) ON DELETE CASCADE,
    user_id    UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    added_by   UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (project_id, user_id)
);
CREATE INDEX idx_rating_project_members_user ON rating_project_members (user_id);
