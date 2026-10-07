-- Whether a report counts toward hiding a post automatically
-- (internal/reports.AutoHideThreshold).
--
-- Every open report used to count. Registration is open, and a new account is
-- held back for its first hours — it cannot post, create a community or upload
-- much — but it could report, so five fresh accounts could hide anyone's
-- public post before a moderator saw it. A report from an account still in
-- its quarantine is kept and reaches the queue exactly as before; it just does
-- not move the automatic threshold.
ALTER TABLE reports ADD COLUMN counts_toward_hide BOOLEAN NOT NULL DEFAULT true;
