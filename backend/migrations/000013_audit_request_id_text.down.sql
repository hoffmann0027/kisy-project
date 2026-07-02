ALTER TABLE audit_logs ALTER COLUMN request_id TYPE UUID USING NULLIF(request_id, '')::uuid;
