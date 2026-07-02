-- HTTP request identifiers (chi middleware.RequestID) are opaque strings,
-- not UUIDs; widen the column so audit rows can store them verbatim.
ALTER TABLE audit_logs ALTER COLUMN request_id TYPE TEXT USING request_id::text;
