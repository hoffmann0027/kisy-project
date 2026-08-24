-- FCM registration tokens of the packaged mobile apps (one device per row).
--
-- Separate from push_subscriptions: a browser subscription is an endpoint URL
-- plus ECDH keys the server encrypts to, while a native device is a single
-- opaque token that Firebase resolves. Both live side by side because one user
-- may have the web app open and the phone app installed at the same time.
CREATE TABLE device_tokens (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id      UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token        TEXT NOT NULL UNIQUE,
    platform     TEXT NOT NULL CHECK (platform IN ('android', 'ios')),
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    last_seen_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_device_tokens_user ON device_tokens (user_id);
