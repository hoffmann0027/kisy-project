-- Role hierarchy: level 1 (CEO, unrestricted) .. level 10 (lowest clearance).
CREATE TABLE roles (
    id          SMALLINT PRIMARY KEY CHECK (id BETWEEN 1 AND 10),
    level       SMALLINT NOT NULL UNIQUE CHECK (level BETWEEN 1 AND 10),
    name        VARCHAR(64) NOT NULL,
    description TEXT,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

COMMENT ON TABLE roles IS 'Fixed 10-level clearance hierarchy. Level 1 = CEO, unrestricted.';

CREATE TABLE permissions (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    code        VARCHAR(64) NOT NULL UNIQUE,
    description TEXT,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE role_permissions (
    role_id       SMALLINT NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    permission_id UUID NOT NULL REFERENCES permissions(id) ON DELETE CASCADE,
    granted_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (role_id, permission_id)
);

CREATE INDEX idx_role_permissions_permission_id ON role_permissions (permission_id);
