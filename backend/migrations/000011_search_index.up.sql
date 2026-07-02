-- Denormalized full-text search index, kept in sync by the application
-- layer whenever a searchable entity (message, user, group) changes.
CREATE TABLE search_index (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    entity_type    VARCHAR(32) NOT NULL CHECK (entity_type IN ('message', 'user', 'group')),
    entity_id      UUID NOT NULL,
    content        TEXT NOT NULL,
    search_vector  TSVECTOR NOT NULL,
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT now(),

    UNIQUE (entity_type, entity_id)
);

CREATE INDEX idx_search_index_vector ON search_index USING GIN (search_vector);

CREATE OR REPLACE FUNCTION search_index_set_vector()
RETURNS TRIGGER AS $$
BEGIN
    NEW.search_vector = to_tsvector('russian', NEW.content);
    NEW.updated_at = now();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_search_index_vector
    BEFORE INSERT OR UPDATE OF content ON search_index
    FOR EACH ROW
    EXECUTE FUNCTION search_index_set_vector();
