ALTER TABLE messages DROP CONSTRAINT IF EXISTS chk_messages_content_or_image;
ALTER TABLE rooms DROP CONSTRAINT IF EXISTS chk_rooms_type;
ALTER TABLE users DROP CONSTRAINT IF EXISTS chk_users_role;
