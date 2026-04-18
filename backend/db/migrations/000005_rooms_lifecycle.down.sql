DROP INDEX IF EXISTS idx_rooms_deleted_at;
ALTER TABLE rooms
    DROP COLUMN IF EXISTS frozen_at,
    DROP COLUMN IF EXISTS deleted_at;
