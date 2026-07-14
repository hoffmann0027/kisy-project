-- Group calendar: one-off events visible to every group member. Board cards
-- with a due_date are surfaced in the calendar separately (no new date here).
CREATE TABLE calendar_events (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    group_id   UUID NOT NULL REFERENCES groups(id) ON DELETE CASCADE,
    title      TEXT NOT NULL,
    starts_at  TIMESTAMPTZ NOT NULL,
    ends_at    TIMESTAMPTZ,
    color      VARCHAR(16) NOT NULL
                   CHECK (color IN ('blue', 'green', 'red', 'orange', 'purple', 'teal', 'pink', 'gray')),
    created_by UUID NOT NULL REFERENCES users(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Month-range lookups per group.
CREATE INDEX idx_calendar_events_group_starts ON calendar_events (group_id, starts_at);

CREATE TRIGGER trg_calendar_events_updated_at
    BEFORE UPDATE ON calendar_events
    FOR EACH ROW
    EXECUTE FUNCTION set_updated_at();
