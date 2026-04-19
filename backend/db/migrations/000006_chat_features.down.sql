DROP TABLE IF EXISTS user_message_bookmarks;
DROP TABLE IF EXISTS room_pins;
DROP TABLE IF EXISTS message_reactions;
DROP TABLE IF EXISTS auth_tokens;

DROP INDEX IF EXISTS idx_messages_parent;
ALTER TABLE messages DROP COLUMN IF EXISTS file_content_type;
ALTER TABLE messages DROP COLUMN IF EXISTS file_name;
ALTER TABLE messages DROP COLUMN IF EXISTS file_url;
ALTER TABLE messages DROP COLUMN IF EXISTS parent_message_id;

ALTER TABLE users DROP COLUMN IF EXISTS email_verified_at;
ALTER TABLE users DROP COLUMN IF EXISTS avatar_url;
