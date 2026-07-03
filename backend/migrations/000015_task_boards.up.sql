-- Trello-style task boards attached to groups. One board per group; the
-- group founder owns the board structure (columns), members manage cards.

CREATE TABLE boards (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    group_id    UUID NOT NULL UNIQUE REFERENCES groups(id) ON DELETE CASCADE,
    title       VARCHAR(128) NOT NULL,
    created_by  UUID NOT NULL REFERENCES users(id),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TRIGGER trg_boards_updated_at
    BEFORE UPDATE ON boards
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TABLE board_columns (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    board_id    UUID NOT NULL REFERENCES boards(id) ON DELETE CASCADE,
    title       VARCHAR(64) NOT NULL,
    position    INTEGER NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_board_columns_board ON board_columns (board_id, position);

CREATE TABLE board_cards (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    board_id    UUID NOT NULL REFERENCES boards(id) ON DELETE CASCADE,
    column_id   UUID NOT NULL REFERENCES board_columns(id) ON DELETE CASCADE,
    title       VARCHAR(256) NOT NULL,
    description TEXT,
    position    INTEGER NOT NULL,
    assignee_id UUID REFERENCES users(id) ON DELETE SET NULL,
    label       VARCHAR(24),
    due_date    TIMESTAMPTZ,
    created_by  UUID NOT NULL REFERENCES users(id),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_board_cards_column ON board_cards (column_id, position);
CREATE INDEX idx_board_cards_board ON board_cards (board_id);
CREATE INDEX idx_board_cards_assignee ON board_cards (assignee_id) WHERE assignee_id IS NOT NULL;

CREATE TRIGGER trg_board_cards_updated_at
    BEFORE UPDATE ON board_cards
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();
