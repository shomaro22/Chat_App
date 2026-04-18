ALTER TABLE rooms ADD COLUMN direct_user_a_id UUID REFERENCES users (id);
ALTER TABLE rooms ADD COLUMN direct_user_b_id UUID REFERENCES users (id);

UPDATE rooms r
SET
    direct_user_a_id = s.small,
    direct_user_b_id = s.large
FROM (
    SELECT m.room_id,
        (array_agg(m.user_id ORDER BY m.user_id))[1] AS small,
        (array_agg(m.user_id ORDER BY m.user_id DESC))[1] AS large
    FROM room_members m
    WHERE m.left_at IS NULL
    GROUP BY m.room_id
    HAVING COUNT(*) = 2
) s
WHERE r.id = s.room_id
  AND r.type = 'direct';

DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM rooms
        WHERE type = 'direct'
          AND (direct_user_a_id IS NULL OR direct_user_b_id IS NULL)
    ) THEN
        RAISE EXCEPTION 'migration 000003: direct room must have exactly two active members to backfill pair columns';
    END IF;
END $$;

ALTER TABLE rooms ADD CONSTRAINT chk_rooms_direct_pair_cols CHECK (
    (type = 'group' AND direct_user_a_id IS NULL AND direct_user_b_id IS NULL)
    OR (
        type = 'direct'
        AND direct_user_a_id IS NOT NULL
        AND direct_user_b_id IS NOT NULL
        AND direct_user_a_id < direct_user_b_id
    )
);

CREATE UNIQUE INDEX uniq_rooms_direct_pair ON rooms (direct_user_a_id, direct_user_b_id)
    WHERE type = 'direct';
