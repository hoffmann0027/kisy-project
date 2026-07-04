-- Store attachment bytes in the database (survives redeploys on ephemeral
-- disks) and record the uploader. message_id becomes nullable so a file can be
-- uploaded before the message that carries it is sent.
ALTER TABLE attachments ADD COLUMN data BYTEA;
ALTER TABLE attachments ADD COLUMN uploaded_by UUID REFERENCES users(id) ON DELETE SET NULL;
ALTER TABLE attachments ALTER COLUMN message_id DROP NOT NULL;
ALTER TABLE attachments ALTER COLUMN storage_path SET DEFAULT '';

CREATE INDEX idx_attachments_unlinked ON attachments (uploaded_by) WHERE message_id IS NULL;
