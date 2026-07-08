-- Promotion conditions: the CEO (level 1) writes the requirement to reach each
-- higher rank. Levels run 1 (CEO, top) … 10 (lowest); a promotion moves a user
-- to a lower number. There are 9 target levels (1..9) a user can be promoted
-- into, hence 9 rules. A member only ever sees the single rule for their own
-- next level (their clearance minus one) — the rest stay hidden so they cannot
-- see the whole ladder in advance. Visibility is enforced in the service.
CREATE TABLE level_conditions (
    target_level INT PRIMARY KEY CHECK (target_level BETWEEN 1 AND 9),
    body         TEXT NOT NULL DEFAULT '',
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_by   UUID REFERENCES users(id) ON DELETE SET NULL
);

-- Seed the 9 rows so the CEO always sees every slot to edit.
INSERT INTO level_conditions (target_level)
SELECT generate_series(1, 9);
