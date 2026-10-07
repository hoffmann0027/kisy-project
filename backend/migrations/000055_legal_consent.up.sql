-- Consent to the privacy policy and the community rules (Google Play UGC
-- policy: rules that define objectionable content must be accepted before a
-- user can publish anything).
--
-- Two places, two jobs:
--   users.privacy_version / rules_version — the CURRENT state, read with every
--     user row, so "does this account still have to accept?" costs no query;
--   legal_acceptances — the EVIDENCE: every acceptance ever given, never
--     updated, so "who agreed to which text, and when" can be
--     answered after the text has changed.
ALTER TABLE users
    ADD COLUMN privacy_version TEXT,
    ADD COLUMN rules_version   TEXT;

CREATE TABLE legal_acceptances (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    document    TEXT NOT NULL CHECK (document IN ('privacy', 'rules')),
    version     TEXT NOT NULL,
    accepted_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    -- Salted digest, like sessions and audit_logs: proof of origin without
    -- keeping the address itself.
    ip_hash     TEXT NOT NULL DEFAULT ''
);

CREATE INDEX idx_legal_acceptances_user ON legal_acceptances (user_id, accepted_at DESC);

-- Evidence is never rewritten. Rows go away only with the user row itself
-- (ON DELETE CASCADE); account deletion anonymises and keeps that row, so the
-- evidence outlives the account like the security journal does.
CREATE FUNCTION legal_acceptances_immutable() RETURNS trigger AS $$
BEGIN
    RAISE EXCEPTION 'legal_acceptances is append-only';
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_legal_acceptances_immutable
    BEFORE UPDATE ON legal_acceptances
    FOR EACH ROW EXECUTE FUNCTION legal_acceptances_immutable();
