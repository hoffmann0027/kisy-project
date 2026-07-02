CREATE TABLE users (
    id                      UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    username                CITEXT NOT NULL UNIQUE,
    display_name            VARCHAR(64) NOT NULL,
    password_hash           TEXT NOT NULL,
    role_id                 SMALLINT NOT NULL REFERENCES roles(id),
    avatar_url              TEXT,
    status                  VARCHAR(16) NOT NULL DEFAULT 'offline'
                                CHECK (status IN ('online', 'offline', 'away')),
    last_seen_at            TIMESTAMPTZ,
    is_active               BOOLEAN NOT NULL DEFAULT true,
    failed_login_attempts   SMALLINT NOT NULL DEFAULT 0,
    locked_until            TIMESTAMPTZ,
    must_change_password    BOOLEAN NOT NULL DEFAULT false,
    created_at              TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at              TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT username_length CHECK (char_length(username::text) BETWEEN 3 AND 32)
);

COMMENT ON TABLE users IS 'Accounts cannot be deleted, only deactivated via is_active.';

CREATE INDEX idx_users_role_id ON users (role_id);
CREATE INDEX idx_users_status ON users (status) WHERE is_active = true;

CREATE TRIGGER trg_users_updated_at
    BEFORE UPDATE ON users
    FOR EACH ROW
    EXECUTE FUNCTION set_updated_at();
