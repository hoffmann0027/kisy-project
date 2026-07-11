-- Stage D (update roadmap): message forwarding.
--
-- forwarded_from_message_id traces the origin for audit (ON DELETE SET NULL
-- so deleting the source does not delete the forward). The sender is
-- DENORMALIZED (id + display name captured at forward time) so the forwarded
-- bubble never reflects the source's *current* visibility or renamed author —
-- it shows who wrote it when it was forwarded, and nothing about where.

ALTER TABLE messages
    ADD COLUMN forwarded_from_message_id UUID REFERENCES messages(id) ON DELETE SET NULL,
    ADD COLUMN forwarded_from_sender_id  UUID,
    ADD COLUMN forwarded_from_sender_name TEXT;
