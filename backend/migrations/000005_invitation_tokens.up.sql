-- Only Level 1 (CEO) creates invitations. Tokens live exactly 120 seconds
-- and are single-use; the plaintext token is never persisted, only its hash.
CREATE TABLE invitation_tokens (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    token_hash  TEXT NOT NULL UNIQUE,
    created_by  UUID NOT NULL REFERENCES users(id),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    expires_at  TIMESTAMPTZ NOT NULL,
    used_at     TIMESTAMPTZ,
    used_by     UUID REFERENCES users(id),

    -- The app must compute created_at itself and insert both columns
    -- explicitly from the same instant (not rely on the column default
    -- plus a separately-computed expiry), otherwise clock skew between
    -- the app and this check will reject the insert.
    CONSTRAINT invitation_ttl_120s CHECK (expires_at = created_at + INTERVAL '120 seconds'),
    CONSTRAINT invitation_usage_consistency CHECK (
        (used_at IS NULL AND used_by IS NULL) OR (used_at IS NOT NULL AND used_by IS NOT NULL)
    )
);

CREATE INDEX idx_invitation_tokens_expires_at ON invitation_tokens (expires_at) WHERE used_at IS NULL;
CREATE INDEX idx_invitation_tokens_created_by ON invitation_tokens (created_by);
