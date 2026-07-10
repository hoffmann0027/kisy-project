-- E2EE infrastructure (docs/e2ee-design.md §6). Additive: plaintext `text`
-- stays until the clients switch to MLS ciphertext, then a follow-up
-- migration drops it (no production data exists yet — design §8).

-- Messages carry MLS ciphertext instead of text once E2EE is on.
ALTER TABLE messages
    ADD COLUMN ciphertext   BYTEA,
    ADD COLUMN alg          SMALLINT,  -- encryption scheme version (1 = MLS suite of design §1)
    ADD COLUMN epoch        BIGINT,    -- MLS epoch, for ordering validation and debugging
    ADD COLUMN content_kind SMALLINT;  -- 1 text, 2 attachment, 3 system; NOT the content itself

-- Ciphertext size cap: an MLS private message for a chat text; large payloads
-- (files) belong in attachments. "Body present" (text/ciphertext/attachment)
-- is enforced by the service layer, not here — attachment-only messages have
-- neither text nor ciphertext.
ALTER TABLE messages ADD CONSTRAINT messages_ciphertext_size CHECK (
    ciphertext IS NULL OR octet_length(ciphertext) <= 65536
);

-- Device directory: one row per device of a user. The Ed25519 identity key
-- is generated on the device and never leaves it; the server only stores the
-- public half. `signed_by`/`signature` is the cross-signing vouch an existing
-- device issues for a newly linked one (design §4). Clients do NOT trust this
-- table blindly: TOFU + safety numbers guard against a lying server (§3.3).
CREATE TABLE e2ee_devices (
    id          UUID PRIMARY KEY,  -- generated client-side at first login
    user_id     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name        VARCHAR(128) NOT NULL DEFAULT '',
    ed25519_pub BYTEA NOT NULL CHECK (octet_length(ed25519_pub) = 32),
    signed_by   UUID REFERENCES e2ee_devices(id) ON DELETE SET NULL,
    signature   BYTEA,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    revoked_at  TIMESTAMPTZ,

    CONSTRAINT e2ee_devices_pub_unique UNIQUE (user_id, ed25519_pub)
);

CREATE INDEX idx_e2ee_devices_user ON e2ee_devices (user_id) WHERE revoked_at IS NULL;

-- One-time MLS key packages ("prekeys"): uploaded in batches by each device,
-- consumed one at a time by whoever adds that device to a chat.
CREATE TABLE e2ee_key_packages (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    device_id   UUID NOT NULL REFERENCES e2ee_devices(id) ON DELETE CASCADE,
    key_package BYTEA NOT NULL CHECK (octet_length(key_package) <= 8192),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    consumed_at TIMESTAMPTZ
);

CREATE INDEX idx_e2ee_key_packages_available ON e2ee_key_packages (device_id) WHERE consumed_at IS NULL;

-- MLS handshake mailbox: Welcome messages target one device; commits and
-- proposals fan out to the whole chat (recipient_device IS NULL). All payloads
-- are ciphertext/protocol frames — the server cannot read group secrets.
CREATE TABLE e2ee_group_messages (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    chat_type        VARCHAR(16) NOT NULL CHECK (chat_type IN ('private', 'group')),
    chat_id          UUID NOT NULL,
    kind             SMALLINT NOT NULL CHECK (kind IN (1, 2, 3)), -- 1 welcome, 2 commit, 3 proposal
    sender_device    UUID REFERENCES e2ee_devices(id) ON DELETE SET NULL,
    recipient_device UUID REFERENCES e2ee_devices(id) ON DELETE CASCADE,
    payload          BYTEA NOT NULL CHECK (octet_length(payload) <= 262144),
    epoch            BIGINT,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    fetched_at       TIMESTAMPTZ
);

CREATE INDEX idx_e2ee_group_messages_chat ON e2ee_group_messages (chat_type, chat_id, created_at);
CREATE INDEX idx_e2ee_group_messages_recipient
    ON e2ee_group_messages (recipient_device) WHERE recipient_device IS NOT NULL AND fetched_at IS NULL;

-- RBAC → crypto bridge (design §5.1): the server records membership changes
-- here; the first online member with commit rights turns each entry into an
-- MLS Add/Remove commit (rotating the group secret) and acknowledges it.
CREATE TABLE e2ee_membership_queue (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    chat_type    VARCHAR(16) NOT NULL CHECK (chat_type IN ('private', 'group')),
    chat_id      UUID NOT NULL,
    action       VARCHAR(16) NOT NULL CHECK (action IN ('add', 'remove')),
    target_user  UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    processed_at TIMESTAMPTZ,
    processed_by UUID REFERENCES e2ee_devices(id) ON DELETE SET NULL
);

CREATE INDEX idx_e2ee_membership_pending
    ON e2ee_membership_queue (chat_type, chat_id) WHERE processed_at IS NULL;

-- Optional encrypted key/history backup (design §4): ciphertext under a key
-- derived from a recovery passphrase with Argon2id ON THE CLIENT. The server
-- stores an opaque blob it cannot decrypt; kdf_params holds only public KDF
-- inputs (salt, opslimit, memlimit) needed to re-derive the key.
CREATE TABLE e2ee_backups (
    user_id    UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    blob       BYTEA NOT NULL CHECK (octet_length(blob) <= 16777216),
    kdf_params JSONB NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
