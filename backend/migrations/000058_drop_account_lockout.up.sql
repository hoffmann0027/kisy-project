-- Audit A-33: failed sign-ins are counted per name and source address in
-- Redis (internal/auth, MaxLoginAttempts), no longer per account. The
-- account-wide counter let anyone who knew a name keep its owner locked out;
-- nothing reads these columns any more.
ALTER TABLE users
    DROP COLUMN failed_login_attempts,
    DROP COLUMN locked_until;
