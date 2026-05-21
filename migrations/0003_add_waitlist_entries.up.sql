CREATE TABLE waitlist_entries (
    id UUID PRIMARY KEY,
    slot_id UUID NOT NULL REFERENCES slots(id) ON DELETE RESTRICT,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    status TEXT NOT NULL CHECK (status IN ('active', 'notified', 'cancelled', 'expired')),
    position BIGSERIAL NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    notified_at TIMESTAMPTZ
);

CREATE UNIQUE INDEX uq_waitlist_slot_user_active_or_notified
    ON waitlist_entries (slot_id, user_id)
    WHERE status IN ('active', 'notified');

CREATE INDEX idx_waitlist_slot_status_position
    ON waitlist_entries (slot_id, status, position);

