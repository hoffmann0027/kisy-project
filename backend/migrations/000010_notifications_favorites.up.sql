CREATE TABLE notifications (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    type        VARCHAR(64) NOT NULL,
    payload     JSONB NOT NULL DEFAULT '{}'::jsonb,
    is_read     BOOLEAN NOT NULL DEFAULT false,
    read_at     TIMESTAMPTZ,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_notifications_unread ON notifications (user_id, created_at DESC) WHERE is_read = false;

CREATE TABLE favorites (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id       UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    chat_type     VARCHAR(16) NOT NULL CHECK (chat_type IN ('private', 'group')),
    chat_id       UUID NOT NULL,
    is_pinned     BOOLEAN NOT NULL DEFAULT false,
    pinned_order  SMALLINT,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),

    UNIQUE (user_id, chat_type, chat_id)
);

CREATE INDEX idx_favorites_user_pinned ON favorites (user_id, pinned_order) WHERE is_pinned = true;
