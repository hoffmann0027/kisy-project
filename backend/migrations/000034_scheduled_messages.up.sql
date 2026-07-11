-- Stage I (UPD3): scheduled message sending.

-- A scheduled message is a frozen SendMessageBody snapshot that the worker
-- replays through the standard send pipeline at send_at. For E2EE private
-- chats the client encrypts at scheduling time ("path A",
-- docs/security.md): the server stores only ciphertext, exactly like a
-- regular E2EE message, and can never read it.
CREATE TABLE scheduled_messages (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    chat_type      VARCHAR(16) NOT NULL CHECK (chat_type IN ('private', 'group')),
    chat_id        UUID NOT NULL,
    sender_id      UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,

    -- Content snapshot: plaintext or E2EE body, never both (same rule as
    -- the messages table).
    text           TEXT CHECK (length(text) <= 8000),
    ciphertext     BYTEA CHECK (octet_length(ciphertext) <= 65536),
    alg            SMALLINT,
    epoch          BIGINT,
    content_kind   SMALLINT,
    reply_to       UUID,
    attachment_ids UUID[] NOT NULL DEFAULT '{}',

    send_at        TIMESTAMPTZ NOT NULL,
    status         VARCHAR(16) NOT NULL DEFAULT 'pending'
        CHECK (status IN ('pending', 'sent', 'canceled')),
    -- The message the worker created (idempotency witness + client-side
    -- plaintext-cache re-keying for E2EE).
    sent_message_id UUID,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT now(),

    CHECK (NOT (text IS NOT NULL AND ciphertext IS NOT NULL))
);

-- The worker's due-scan touches only pending rows.
CREATE INDEX idx_scheduled_messages_due ON scheduled_messages (send_at)
    WHERE status = 'pending';
CREATE INDEX idx_scheduled_messages_sender ON scheduled_messages (sender_id, status, send_at);

-- Marks a message as born from the scheduler. Metadata only (like
-- reply_to): lets the sender's client re-key its locally cached plaintext
-- from sched/<scheduledId> to msg/<messageId> — MLS senders consume their
-- keys at encryption time and cannot decrypt their own ciphertext later.
ALTER TABLE messages ADD COLUMN scheduled_message_id UUID;
