ALTER TABLE rooms
    ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMPTZ,
    ADD COLUMN IF NOT EXISTS frozen_at TIMESTAMPTZ;

CREATE INDEX IF NOT EXISTS idx_rooms_deleted_at ON rooms (deleted_at) WHERE deleted_at IS NOT NULL;
