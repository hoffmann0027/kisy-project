-- The language each account's app shows (internal/locales), so a push written
-- by the server reaches them in it. Russian for everyone until their app says
-- otherwise: every app before this one was Russian.
ALTER TABLE users ADD COLUMN locale text NOT NULL DEFAULT 'ru'
    CONSTRAINT users_locale_format CHECK (locale ~ '^[a-z]{2}$');
