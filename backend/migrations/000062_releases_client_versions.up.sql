-- "New Update" in the admin panel (October 2026): the CEO announces a new
-- version of the app to everyone, and sees how many people still run an
-- older one.

-- Each announcement of a version: what it is called, what changed, where to
-- get it. The notification itself goes to every active account (type
-- app_release, internal/announcements).
CREATE TABLE app_releases (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    version         VARCHAR(32) NOT NULL CHECK (length(btrim(version)) > 0),
    notes           VARCHAR(4000) NOT NULL CHECK (length(btrim(notes)) > 0),
    download_url    TEXT CHECK (download_url IS NULL OR download_url LIKE 'https://%'),
    recipient_count INTEGER NOT NULL DEFAULT 0,
    created_by      UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_app_releases_created ON app_releases (created_at DESC);

-- Which build of the app each account last used, per platform. The app
-- reports it in X-Kisy-App-Version; the server writes it at most every few
-- hours per account and build (internal/clientversions).
CREATE TABLE client_versions (
    user_id      UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    platform     VARCHAR(16) NOT NULL CHECK (platform IN ('android')),
    version      VARCHAR(64) NOT NULL,
    build        BIGINT NOT NULL CHECK (build >= 0),
    last_seen_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (user_id, platform)
);
CREATE INDEX idx_client_versions_seen ON client_versions (platform, last_seen_at DESC);
