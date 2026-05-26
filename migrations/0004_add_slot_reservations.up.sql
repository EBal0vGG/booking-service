CREATE TABLE slot_reservations (
    id UUID PRIMARY KEY,
    slot_id UUID NOT NULL REFERENCES slots(id) ON DELETE RESTRICT,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    waitlist_entry_id UUID REFERENCES waitlist_entries(id) ON DELETE RESTRICT,
    status TEXT NOT NULL CHECK (status IN ('active', 'confirmed', 'expired', 'cancelled')),
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    confirmed_at TIMESTAMPTZ,
    expired_at TIMESTAMPTZ
);

CREATE UNIQUE INDEX uq_slot_reservations_active_slot
    ON slot_reservations(slot_id)
    WHERE status = 'active';

CREATE INDEX idx_slot_reservations_expired
    ON slot_reservations(status, expires_at)
    WHERE status = 'active';

CREATE INDEX idx_slot_reservations_user_status
    ON slot_reservations(user_id, status);
