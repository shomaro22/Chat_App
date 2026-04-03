CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR NOT NULL,
    email VARCHAR NOT NULL UNIQUE,
    password_hash VARCHAR NOT NULL,
    role VARCHAR NOT NULL,
    active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE rooms (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    type VARCHAR NOT NULL,
    name VARCHAR,
    created_by UUID NOT NULL REFERENCES users (id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_rooms_type_created_at ON rooms (type, created_at);
CREATE INDEX idx_rooms_created_by_created_at ON rooms (created_by, created_at);

CREATE TABLE room_members (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    room_id UUID NOT NULL REFERENCES rooms (id),
    user_id UUID NOT NULL REFERENCES users (id),
    joined_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    left_at TIMESTAMPTZ,
    CONSTRAINT uniq_room_members_room_user UNIQUE (room_id, user_id)
);

CREATE INDEX idx_room_members_user_left_at ON room_members (user_id, left_at);
CREATE INDEX idx_room_members_room_left_at ON room_members (room_id, left_at);

CREATE TABLE messages (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    room_id UUID NOT NULL REFERENCES rooms (id),
    sender_id UUID NOT NULL REFERENCES users (id),
    content TEXT,
    image_url TEXT,
    is_deleted BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at TIMESTAMPTZ
);

CREATE INDEX idx_messages_room_created_id ON messages (room_id, created_at, id);
CREATE INDEX idx_messages_sender_created_at ON messages (sender_id, created_at);
CREATE INDEX idx_messages_room_deleted_created_id ON messages (room_id, is_deleted, created_at, id);

CREATE TABLE message_reads (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    room_id UUID NOT NULL REFERENCES rooms (id),
    user_id UUID NOT NULL REFERENCES users (id),
    read_message_id UUID REFERENCES messages (id),
    read_at TIMESTAMPTZ,
    CONSTRAINT uniq_message_reads_room_user UNIQUE (room_id, user_id)
);

CREATE INDEX idx_message_reads_user_read_at ON message_reads (user_id, read_at);
