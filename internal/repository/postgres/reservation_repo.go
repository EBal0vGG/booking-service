package postgres

import (
	"context"
	"errors"
	"time"

	"booking-service/internal/domain"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ReservationRepo struct {
	pool *pgxpool.Pool
}

func NewReservationRepo(pool *pgxpool.Pool) *ReservationRepo {
	return &ReservationRepo{pool: pool}
}

func (r *ReservationRepo) Create(ctx context.Context, reservation domain.SlotReservation) (*domain.SlotReservation, error) {
	const q = `
INSERT INTO slot_reservations (
    id, slot_id, user_id, waitlist_entry_id, status, expires_at, created_at, confirmed_at, expired_at
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
RETURNING id, slot_id, user_id, waitlist_entry_id, status, expires_at, created_at, confirmed_at, expired_at
`
	db := dbFromContext(ctx, r.pool)
	row := db.QueryRow(
		ctx,
		q,
		reservation.ID,
		reservation.SlotID,
		reservation.UserID,
		reservation.WaitlistEntryID,
		reservation.Status,
		reservation.ExpiresAt,
		reservation.CreatedAt,
		reservation.ConfirmedAt,
		reservation.ExpiredAt,
	)
	created, err := scanSlotReservation(row)
	if err != nil {
		if isReservationDuplicate(err) {
			return nil, domain.NewDomainError(domain.ErrorSlotReserved, "slot is already reserved")
		}
		return nil, err
	}
	return created, nil
}

func (r *ReservationRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.SlotReservation, error) {
	const q = `
SELECT id, slot_id, user_id, waitlist_entry_id, status, expires_at, created_at, confirmed_at, expired_at
FROM slot_reservations
WHERE id = $1
`
	db := dbFromContext(ctx, r.pool)
	row := db.QueryRow(ctx, q, id)
	reservation, err := scanSlotReservation(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return reservation, nil
}

func (r *ReservationRepo) GetByIDForUpdate(ctx context.Context, id uuid.UUID) (*domain.SlotReservation, error) {
	const q = `
SELECT id, slot_id, user_id, waitlist_entry_id, status, expires_at, created_at, confirmed_at, expired_at
FROM slot_reservations
WHERE id = $1
FOR UPDATE
`
	db := dbFromContext(ctx, r.pool)
	row := db.QueryRow(ctx, q, id)
	reservation, err := scanSlotReservation(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return reservation, nil
}

func (r *ReservationRepo) ListActiveByUser(ctx context.Context, userID uuid.UUID, now time.Time) ([]domain.SlotReservation, error) {
	const q = `
SELECT id, slot_id, user_id, waitlist_entry_id, status, expires_at, created_at, confirmed_at, expired_at
FROM slot_reservations
WHERE user_id = $1
  AND status = 'active'
  AND expires_at > $2
ORDER BY expires_at ASC
`
	db := dbFromContext(ctx, r.pool)
	rows, err := db.Query(ctx, q, userID, now.UTC())
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	reservations := make([]domain.SlotReservation, 0)
	for rows.Next() {
		reservation, scanErr := scanSlotReservation(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		reservations = append(reservations, *reservation)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return reservations, nil
}

func (r *ReservationRepo) GetActiveBySlot(ctx context.Context, slotID uuid.UUID) (*domain.SlotReservation, error) {
	const q = `
SELECT id, slot_id, user_id, waitlist_entry_id, status, expires_at, created_at, confirmed_at, expired_at
FROM slot_reservations
WHERE slot_id = $1
  AND status = 'active'
`
	db := dbFromContext(ctx, r.pool)
	row := db.QueryRow(ctx, q, slotID)
	reservation, err := scanSlotReservation(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return reservation, nil
}

func (r *ReservationRepo) GetActiveBySlotForUpdate(ctx context.Context, slotID uuid.UUID) (*domain.SlotReservation, error) {
	const q = `
SELECT id, slot_id, user_id, waitlist_entry_id, status, expires_at, created_at, confirmed_at, expired_at
FROM slot_reservations
WHERE slot_id = $1
  AND status = 'active'
FOR UPDATE
`
	db := dbFromContext(ctx, r.pool)
	row := db.QueryRow(ctx, q, slotID)
	reservation, err := scanSlotReservation(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return reservation, nil
}

func (r *ReservationRepo) SetConfirmed(ctx context.Context, id uuid.UUID, confirmedAt time.Time) (*domain.SlotReservation, error) {
	const q = `
UPDATE slot_reservations
SET status = 'confirmed',
    confirmed_at = $2
WHERE id = $1
  AND status = 'active'
RETURNING id, slot_id, user_id, waitlist_entry_id, status, expires_at, created_at, confirmed_at, expired_at
`
	db := dbFromContext(ctx, r.pool)
	row := db.QueryRow(ctx, q, id, confirmedAt.UTC())
	reservation, err := scanSlotReservation(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return r.GetByID(ctx, id)
	}
	if err != nil {
		return nil, err
	}
	return reservation, nil
}

func (r *ReservationRepo) SetCancelled(ctx context.Context, id uuid.UUID) (*domain.SlotReservation, error) {
	const q = `
UPDATE slot_reservations
SET status = 'cancelled'
WHERE id = $1
  AND status = 'active'
RETURNING id, slot_id, user_id, waitlist_entry_id, status, expires_at, created_at, confirmed_at, expired_at
`
	db := dbFromContext(ctx, r.pool)
	row := db.QueryRow(ctx, q, id)
	reservation, err := scanSlotReservation(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return r.GetByID(ctx, id)
	}
	if err != nil {
		return nil, err
	}
	return reservation, nil
}

func (r *ReservationRepo) ExpireBatch(ctx context.Context, now time.Time, limit int) ([]domain.SlotReservation, error) {
	const q = `
WITH picked AS (
    SELECT id
    FROM slot_reservations
    WHERE status = 'active'
      AND expires_at <= $1
    ORDER BY expires_at
    LIMIT $2
    FOR UPDATE SKIP LOCKED
)
UPDATE slot_reservations r
SET status = 'expired',
    expired_at = $1
FROM picked
WHERE r.id = picked.id
RETURNING r.id, r.slot_id, r.user_id, r.waitlist_entry_id, r.status, r.expires_at, r.created_at, r.confirmed_at, r.expired_at
`
	db := dbFromContext(ctx, r.pool)
	rows, err := db.Query(ctx, q, now.UTC(), limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	reservations := make([]domain.SlotReservation, 0, limit)
	for rows.Next() {
		reservation, scanErr := scanSlotReservation(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		reservations = append(reservations, *reservation)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return reservations, nil
}

type reservationScanner interface {
	Scan(dest ...any) error
}

func scanSlotReservation(row reservationScanner) (*domain.SlotReservation, error) {
	var reservation domain.SlotReservation
	if err := row.Scan(
		&reservation.ID,
		&reservation.SlotID,
		&reservation.UserID,
		&reservation.WaitlistEntryID,
		&reservation.Status,
		&reservation.ExpiresAt,
		&reservation.CreatedAt,
		&reservation.ConfirmedAt,
		&reservation.ExpiredAt,
	); err != nil {
		return nil, err
	}
	return &reservation, nil
}

func isReservationDuplicate(err error) bool {
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) {
		return false
	}
	return pgErr.Code == "23505" && pgErr.ConstraintName == "uq_slot_reservations_active_slot"
}
