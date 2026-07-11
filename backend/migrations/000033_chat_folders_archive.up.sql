-- Stage H (UPD3): per-user chat folders + chat archive.

-- Folders are personal organizational metadata: named, ordered containers
-- of chat references. They never grant or reveal access — the chat list is
-- always produced by the access-filtered chats/groups services, so a
-- folder item pointing at an inaccessible chat simply never surfaces.
CREATE TABLE chat_folders (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name       VARCHAR(64) NOT NULL,
    position   INT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_chat_folders_user ON chat_folders (user_id, position);

-- A folder item is just a reference; content stays in chats/groups.
CREATE TABLE chat_folder_items (
    folder_id  UUID NOT NULL REFERENCES chat_folders(id) ON DELETE CASCADE,
    chat_type  VARCHAR(16) NOT NULL CHECK (chat_type IN ('private', 'group')),
    chat_id    UUID NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),

    PRIMARY KEY (folder_id, chat_type, chat_id)
);

-- Archive is personal (like chat_mutes, stage G): user X's archive is
-- visible only to X. Not a column on chats/groups — those are shared.
CREATE TABLE chat_archives (
    user_id     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    chat_type   VARCHAR(16) NOT NULL CHECK (chat_type IN ('private', 'group')),
    chat_id     UUID NOT NULL,
    archived_at TIMESTAMPTZ NOT NULL DEFAULT now(),

    PRIMARY KEY (user_id, chat_type, chat_id)
);
