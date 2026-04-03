ALTER TABLE users
    ADD CONSTRAINT chk_users_role CHECK (role IN ('user', 'admin'));

ALTER TABLE rooms
    ADD CONSTRAINT chk_rooms_type CHECK (type IN ('direct', 'group'));

ALTER TABLE messages
    ADD CONSTRAINT chk_messages_content_or_image CHECK (
        is_deleted
        OR (
            (content IS NOT NULL AND btrim(content) <> '')
            OR (image_url IS NOT NULL AND btrim(image_url) <> '')
        )
    );
