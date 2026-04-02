CREATE EXTENSION IF NOT EXISTS btree_gist;

CREATE TABLE users (
    id UUID PRIMARY KEY,
    email TEXT NOT NULL UNIQUE,
    role TEXT NOT NULL CHECK (role IN ('admin', 'user')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE rooms (
    id UUID PRIMARY KEY,
    name TEXT NOT NULL,
    description TEXT,
    capacity INT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE schedules (
    id UUID PRIMARY KEY,
    room_id UUID NOT NULL REFERENCES rooms(id) ON DELETE CASCADE,
    day_of_week INT NOT NULL CHECK (day_of_week BETWEEN 1 AND 7),
    start_time TIME NOT NULL,
    end_time TIME NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CHECK (start_time < end_time),
    UNIQUE (room_id, day_of_week)
);

CREATE INDEX idx_schedules_room_id ON schedules (room_id);

CREATE TABLE slots (
    id UUID PRIMARY KEY,
    room_id UUID NOT NULL REFERENCES rooms(id) ON DELETE CASCADE,
    start_time TIMESTAMPTZ NOT NULL,
    end_time TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CHECK (start_time < end_time),
    UNIQUE (room_id, start_time)
);

ALTER TABLE slots
    ADD CONSTRAINT slots_no_overlap
    EXCLUDE USING gist (
        room_id WITH =,
        tstzrange(start_time, end_time, '[)') WITH &&
    );

CREATE INDEX idx_slots_room_id_start_time ON slots (room_id, start_time);

CREATE TABLE bookings (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    slot_id UUID NOT NULL REFERENCES slots(id) ON DELETE RESTRICT,
    status TEXT NOT NULL CHECK (status IN ('active', 'cancelled')),
    conference_link TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX uq_bookings_active_slot
    ON bookings (slot_id)
    WHERE status = 'active';

CREATE INDEX idx_bookings_user_id ON bookings (user_id);
CREATE INDEX idx_bookings_user_active ON bookings (user_id) WHERE status = 'active';
CREATE INDEX idx_bookings_slot_id ON bookings (slot_id);
