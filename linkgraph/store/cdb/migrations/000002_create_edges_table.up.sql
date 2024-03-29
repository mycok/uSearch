CREATE TABLE IF NOT EXISTS edges (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    src UUID NOT NULL REFERENCES links(id) ON DELETE CASCADE,
    dest UUID NOT NULL REFERENCES links(id) ON DELETE CASCADE,
    updated_at timestamptz,
    CONSTRAINT edge_links UNIQUE(src, dest)
);
