DROP INDEX IF EXISTS uniq_rooms_direct_pair;
ALTER TABLE rooms DROP CONSTRAINT IF EXISTS chk_rooms_direct_pair_cols;
ALTER TABLE rooms DROP COLUMN IF EXISTS direct_user_b_id;
ALTER TABLE rooms DROP COLUMN IF EXISTS direct_user_a_id;
