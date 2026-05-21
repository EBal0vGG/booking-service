package postgres

import (
	"context"
	"time"

	"booking-service/internal/domain"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type BookingRepo struct {
	pool *pgxpool.Pool
}

func NewBookingRepo(pool *pgxpool.Pool) *BookingRepo {
	return &BookingRepo{pool: pool}
}

func (r *BookingRepo) Create(ctx context.Context, booking domain.Booking) error {
	const q = `
INSERT INTO bookings (id, user_id, slot_id, status, conference_link, created_at)
VALUES ($1, $2, $3, $4, $5, $6)
ON CONFLICT (slot_id) WHERE (status = 'active') DO NOTHING
`
	db := dbFromContext(ctx, r.pool)
	tag, err := db.Exec(ctx, q, booking.ID, booking.UserID, booking.SlotID, booking.Status, booking.ConferenceLink, booking.CreatedAt)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		// Alternative: return a sentinel infra error and map to SLOT_ALREADY_BOOKED in usecase.
		return domain.NewDomainError(domain.ErrorSlotAlreadyBooked, "slot already booked")
	}
	return nil
}

func (r *BookingRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.Booking, error) {
	const q = `
SELECT id, user_id, slot_id, status, conference_link, created_at
FROM bookings
WHERE id = $1
`
	db := dbFromContext(ctx, r.pool)
	var b domain.Booking
	var conferenceLink pgtype.Text
	if err := db.QueryRow(ctx, q, id).Scan(&b.ID, &b.UserID, &b.SlotID, &b.Status, &conferenceLink, &b.CreatedAt); err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	if conferenceLink.Valid {
		v := conferenceLink.String
		b.ConferenceLink = &v
	}
	return &b, nil
}

func (r *BookingRepo) HasActiveBySlot(ctx context.Context, slotID uuid.UUID) (bool, error) {
	const q = `
SELECT EXISTS(
    SELECT 1
    FROM bookings
    WHERE slot_id = $1
      AND status = 'active'
)
`
	db := dbFromContext(ctx, r.pool)
	var exists bool
	if err := db.QueryRow(ctx, q, slotID).Scan(&exists); err != nil {
		return false, err
	}
	return exists, nil
}

func (r *BookingRepo) GetActiveBySlotForUpdate(ctx context.Context, slotID uuid.UUID) (*domain.Booking, error) {
	const q = `
SELECT id, user_id, slot_id, status, conference_link, created_at
FROM bookings
WHERE slot_id = $1
  AND status = 'active'
FOR UPDATE
`
	db := dbFromContext(ctx, r.pool)
	var b domain.Booking
	var conferenceLink pgtype.Text
	if err := db.QueryRow(ctx, q, slotID).Scan(&b.ID, &b.UserID, &b.SlotID, &b.Status, &conferenceLink, &b.CreatedAt); err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	if conferenceLink.Valid {
		v := conferenceLink.String
		b.ConferenceLink = &v
	}
	return &b, nil
}

func (r *BookingRepo) SetCancelled(ctx context.Context, id uuid.UUID) (*domain.Booking, error) {
	const q = `
UPDATE bookings
SET status = 'cancelled'
WHERE id = $1 AND status = 'active'
RETURNING id, user_id, slot_id, status, conference_link, created_at
`
	db := dbFromContext(ctx, r.pool)
	var b domain.Booking
	var conferenceLink pgtype.Text
	if err := db.QueryRow(ctx, q, id).Scan(&b.ID, &b.UserID, &b.SlotID, &b.Status, &conferenceLink, &b.CreatedAt); err != nil {
		if err == pgx.ErrNoRows {
			return r.GetByID(ctx, id)
		}
		return nil, err
	}
	if conferenceLink.Valid {
		v := conferenceLink.String
		b.ConferenceLink = &v
	}
	return &b, nil
}

func (r *BookingRepo) List(ctx context.Context, page, pageSize int) ([]domain.Booking, int, error) {
	const qCount = `SELECT count(*) FROM bookings`
	const qList = `
SELECT id, user_id, slot_id, status, conference_link, created_at
FROM bookings
ORDER BY created_at DESC
LIMIT $1 OFFSET $2
`
	db := dbFromContext(ctx, r.pool)

	var total int
	if err := db.QueryRow(ctx, qCount).Scan(&total); err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	rows, err := db.Query(ctx, qList, pageSize, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	out := make([]domain.Booking, 0, pageSize)
	for rows.Next() {
		var b domain.Booking
		var conferenceLink pgtype.Text
		if err := rows.Scan(&b.ID, &b.UserID, &b.SlotID, &b.Status, &conferenceLink, &b.CreatedAt); err != nil {
			return nil, 0, err
		}
		if conferenceLink.Valid {
			v := conferenceLink.String
			b.ConferenceLink = &v
		}
		out = append(out, b)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	return out, total, nil
}

func (r *BookingRepo) ListFutureByUser(ctx context.Context, userID uuid.UUID, now time.Time) ([]domain.Booking, error) {
	const q = `
SELECT b.id, b.user_id, b.slot_id, b.status, b.conference_link, b.created_at
FROM bookings b
JOIN slots s ON s.id = b.slot_id
WHERE b.user_id = $1
  AND b.status = 'active'
  AND s.start_time >= $2
ORDER BY s.start_time ASC
`
	db := dbFromContext(ctx, r.pool)
	rows, err := db.Query(ctx, q, userID, now.UTC())
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []domain.Booking
	for rows.Next() {
		var b domain.Booking
		var conferenceLink pgtype.Text
		if err := rows.Scan(&b.ID, &b.UserID, &b.SlotID, &b.Status, &conferenceLink, &b.CreatedAt); err != nil {
			return nil, err
		}
		if conferenceLink.Valid {
			v := conferenceLink.String
			b.ConferenceLink = &v
		}
		out = append(out, b)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}
