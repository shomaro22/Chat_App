ALTER TABLE message_reads
    ADD COLUMN IF NOT EXISTS created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW();

UPDATE message_reads
SET
    created_at = COALESCE(read_at, created_at),
    updated_at = COALESCE(read_at, updated_at);
