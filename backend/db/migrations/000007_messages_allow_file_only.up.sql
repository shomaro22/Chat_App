ALTER TABLE messages DROP CONSTRAINT IF EXISTS chk_messages_content_or_image;

ALTER TABLE messages
    ADD CONSTRAINT chk_messages_content_or_image_or_file CHECK (
        is_deleted
        OR (
            (content IS NOT NULL AND btrim(content) <> '')
            OR (image_url IS NOT NULL AND btrim(image_url) <> '')
            OR (file_url IS NOT NULL AND btrim(file_url) <> '')
        )
    );
