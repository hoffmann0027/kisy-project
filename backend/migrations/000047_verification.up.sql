-- The verification mark, for people and for groups/communities.
--
-- Given and taken away by the CEO only (internal/admin). A timestamp rather
-- than a boolean: "verified since" is what the audit log and the admin screen
-- show, and NULL is the unverified state. verified_by says who gave it; the
-- audit log keeps the full history of giving and taking away.
ALTER TABLE users
    ADD COLUMN verified_at TIMESTAMPTZ,
    ADD COLUMN verified_by UUID REFERENCES users(id) ON DELETE SET NULL;

ALTER TABLE groups
    ADD COLUMN verified_at TIMESTAMPTZ,
    ADD COLUMN verified_by UUID REFERENCES users(id) ON DELETE SET NULL;

-- Both come together or not at all.
ALTER TABLE users ADD CONSTRAINT users_verified_by_with_at
    CHECK (verified_by IS NULL OR verified_at IS NOT NULL);
ALTER TABLE groups ADD CONSTRAINT groups_verified_by_with_at
    CHECK (verified_by IS NULL OR verified_at IS NOT NULL);
