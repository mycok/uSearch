CREATE TABLE IF NOT EXISTS links (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    url STRING NOT NULL UNIQUE,
    retrieved_at timestamptz
);
