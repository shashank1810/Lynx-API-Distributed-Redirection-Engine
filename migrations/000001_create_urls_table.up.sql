-- Create the core URLs table with indexing for high-throughput lookups.

CREATE TABLE IF NOT EXISTS urls (
    id          BIGSERIAL PRIMARY KEY,
    short_code  VARCHAR(16) NOT NULL UNIQUE,
    original_url TEXT NOT NULL,
    clicks      BIGINT NOT NULL DEFAULT 0,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at  TIMESTAMPTZ,
    is_active   BOOLEAN NOT NULL DEFAULT TRUE,
    created_by  VARCHAR(255)
);

-- Index for the hot-path: resolve short_code → original_url.
CREATE INDEX IF NOT EXISTS idx_urls_short_code ON urls (short_code) WHERE is_active = TRUE;

-- Index for expiration cleanup jobs.
CREATE INDEX IF NOT EXISTS idx_urls_expires_at ON urls (expires_at) WHERE expires_at IS NOT NULL AND is_active = TRUE;

-- Trigger to auto-update the updated_at timestamp on row modification.
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_urls_updated_at
    BEFORE UPDATE ON urls
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();
