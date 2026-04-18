ALTER TABLE message_reads
    DROP COLUMN IF EXISTS updated_at,
    DROP COLUMN IF EXISTS created_at;
