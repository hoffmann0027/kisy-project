-- Who may not come back to a group or community. A ban removes the member and
-- keeps them out: "Вступить", a join request and being added are all refused
-- while the row exists. Founders, owners and their editors and moderators
-- ban; the same people lift it.
CREATE TABLE group_bans (
    group_id   UUID NOT NULL REFERENCES groups(id) ON DELETE CASCADE,
    user_id    UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    banned_by  UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (group_id, user_id)
);
