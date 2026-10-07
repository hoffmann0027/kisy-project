DROP TRIGGER IF EXISTS trg_legal_acceptances_immutable ON legal_acceptances;
DROP FUNCTION IF EXISTS legal_acceptances_immutable();
DROP TABLE IF EXISTS legal_acceptances;
ALTER TABLE users
    DROP COLUMN IF EXISTS privacy_version,
    DROP COLUMN IF EXISTS rules_version;
