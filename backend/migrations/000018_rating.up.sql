-- "Рейтинг" project board: projects (backlog) → tasks (in progress) → done,
-- with a per-project net-profit ledger. Money is stored as integer kopecks
-- (never floating point).

CREATE TABLE rating_projects (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    title       TEXT NOT NULL,
    description TEXT,
    difficulty  TEXT NOT NULL DEFAULT 'medium' CHECK (difficulty IN ('easy', 'medium', 'hard')),
    created_by  UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE rating_tasks (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id  UUID NOT NULL REFERENCES rating_projects(id) ON DELETE CASCADE,
    title       TEXT NOT NULL,
    -- assignee is null while the task sits in the backlog; a user may only
    -- assign themselves, which moves the task to 'in_progress'.
    assignee_id UUID REFERENCES users(id) ON DELETE SET NULL,
    progress    INT NOT NULL DEFAULT 0 CHECK (progress BETWEEN 0 AND 100),
    status      TEXT NOT NULL DEFAULT 'backlog' CHECK (status IN ('backlog', 'in_progress', 'done')),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_rating_tasks_project ON rating_tasks (project_id);
CREATE INDEX idx_rating_tasks_status ON rating_tasks (status);

-- Append-only ledger: "прибыль за всё время" is the sum of these rows, so a
-- new entry adds to history rather than overwriting a single mutable number.
CREATE TABLE rating_finance_entries (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id      UUID NOT NULL REFERENCES rating_projects(id) ON DELETE CASCADE,
    task_id         UUID REFERENCES rating_tasks(id) ON DELETE SET NULL,
    income_kopecks  BIGINT NOT NULL DEFAULT 0 CHECK (income_kopecks >= 0),
    expense_kopecks BIGINT NOT NULL DEFAULT 0 CHECK (expense_kopecks >= 0),
    note            TEXT,
    created_by      UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_rating_finance_project ON rating_finance_entries (project_id);
CREATE INDEX idx_rating_finance_task ON rating_finance_entries (task_id);
CREATE INDEX idx_rating_finance_created_at ON rating_finance_entries (created_at);
