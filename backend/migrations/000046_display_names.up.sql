-- Unique, letters-only display names.
--
-- username needs nothing here: it has been CITEXT UNIQUE since migration 3,
-- which is already unique without regard to case.
--
-- display_name had no rule at all — registration even copied the username
-- into it, underscores and digits included. The new rule: letters of the
-- Latin and Cyrillic alphabets, single spaces between words, 2–40 characters,
-- unique ignoring case and repeated spaces.
--
-- Existing names are NOT rewritten. A name that breaks the rule, or that
-- collides with an earlier account's name, is flagged instead, and its owner
-- is made to choose a new one at their next sign-in. Uniqueness is enforced
-- right away for everyone who is not flagged; once nobody is flagged,
-- scripts/finalize-display-names.sh makes it unconditional.

-- The comparison key, in one place. Computed by the database rather than by
-- Go so that the stored key and the key a search or an insert compares against
-- can never be two implementations of the same idea.
--
-- lower() folds ASCII the same way under every locale, but Cyrillic only under
-- a Unicode-aware one — and the production database's locale is not something
-- this file controls. The explicit translate() folds Cyrillic regardless.
CREATE FUNCTION kisy_display_name_key(name text) RETURNS text
LANGUAGE sql IMMUTABLE STRICT PARALLEL SAFE
AS $$
    SELECT lower(translate(
        regexp_replace(btrim(name), '\s+', ' ', 'g'),
        'АБВГДЕЁЖЗИЙКЛМНОПРСТУФХЦЧШЩЪЫЬЭЮЯЄІЇҐЎЂЃЅЈЉЊЋЌЏ',
        'абвгдеёжзийклмнопрстуфхцчшщъыьэюяєіїґўђѓѕјљњћќџ'
    ))
$$;

-- The same alphabet the backend accepts (internal/users/displayname.go):
-- Latin letters including Latin-1 and Latin Extended-A/B, and the Cyrillic
-- block minus its non-letters (U+0482–U+0489: a numeric sign and combining
-- marks). A word is one or more letters; words are separated by one space.
CREATE FUNCTION kisy_display_name_valid(name text) RETURNS boolean
LANGUAGE sql IMMUTABLE STRICT PARALLEL SAFE
AS $$
    SELECT char_length(name) BETWEEN 2 AND 40
       AND name ~ '^[A-Za-zÀ-ÖØ-öø-ɏЀ-ҁҊ-ӿ]+( [A-Za-zÀ-ÖØ-öø-ɏЀ-ҁҊ-ӿ]+)*$'
$$;

ALTER TABLE users
    ADD COLUMN display_name_needs_change BOOLEAN NOT NULL DEFAULT false,
    ADD COLUMN display_name_key TEXT GENERATED ALWAYS AS (kisy_display_name_key(display_name)) STORED;

-- 1. Names that break the rule.
UPDATE users SET display_name_needs_change = true
WHERE NOT kisy_display_name_valid(display_name);

-- 2. Valid names that collide. The earliest account keeps the name; everyone
--    who took it later chooses again. Ties on created_at break by id, so
--    exactly one holder remains per name.
WITH ranked AS (
    SELECT id, row_number() OVER (PARTITION BY display_name_key ORDER BY created_at, id) AS rn
    FROM users
    WHERE NOT display_name_needs_change
)
UPDATE users u SET display_name_needs_change = true
FROM ranked r
WHERE u.id = r.id AND r.rn > 1;

-- Unique among everyone who is not flagged. A flagged account is excluded
-- until it picks a name, and the UPDATE that sets the name also clears the
-- flag, so a new name that collides is refused by this index in that same
-- statement.
CREATE UNIQUE INDEX uq_users_display_name_key ON users (display_name_key)
    WHERE NOT display_name_needs_change;

-- The rule itself, for every account that is not flagged. Valid at once:
-- every row either passes or carries the flag.
ALTER TABLE users ADD CONSTRAINT users_display_name_valid
    CHECK (display_name_needs_change OR kisy_display_name_valid(display_name));

-- The basic-account lookup compares on the key (internal/users/repository.go).
-- The unique index already covers unflagged rows; flagged ones are few.
