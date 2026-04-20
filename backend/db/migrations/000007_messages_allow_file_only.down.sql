ALTER TABLE messages DROP CONSTRAINT IF EXISTS chk_messages_content_or_image_or_file;

ALTER TABLE messages
    ADD CONSTRAINT chk_messages_content_or_image CHECK (
        is_deleted
        OR (
            (content IS NOT NULL AND btrim(content) <> '')
            OR (image_url IS NOT NULL AND btrim(image_url) <> '')
        )
    );
