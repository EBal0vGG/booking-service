package postgres

import (
	"context"
	"errors"

	"booking-service/internal/domain"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type WaitlistRepo struct {
	pool *pgxpool.Pool
}

func NewWaitlistRepo(pool *pgxpool.Pool) *WaitlistRepo {
	return &WaitlistRepo{pool: pool}
}

func (r *WaitlistRepo) Join(ctx context.Context, entry domain.WaitlistEntry) (*domain.WaitlistEntry, error) {
	const q = `
INSERT INTO waitlist_entries (id, slot_id, user_id, status, created_at)
VALUES ($1, $2, $3, $4, $5)
RETURNING id, slot_id, user_id, status, position, created_at, notified_at
`
	db := dbFromContext(ctx, r.pool)
	row := db.QueryRow(ctx, q, entry.ID, entry.SlotID, entry.UserID, entry.Status, entry.CreatedAt)
	joined, err := scanWaitlistEntry(row)
	if err != nil {
		if isWaitlistDuplicateJoin(err) {
			return nil, domain.NewDomainError(domain.ErrorWaitlistJoined, "user is already in waitlist for this slot")
		}
		return nil, err
	}
	return joined, nil
}

func (r *WaitlistRepo) Leave(ctx context.Context, entryID, userID uuid.UUID) (*domain.WaitlistEntry, bool, error) {
	const q = `
UPDATE waitlist_entries
SET status = 'cancelled'
WHERE id = $1
  AND user_id = $2
  AND status IN ('active', 'notified')
RETURNING id, slot_id, user_id, status, position, created_at, notified_at
`
	db := dbFromContext(ctx, r.pool)
	row := db.QueryRow(ctx, q, entryID, userID)
	updated, err := scanWaitlistEntry(row)
	if err == nil {
		return updated, true, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return nil, false, err
	}
	entry, err := r.getByIDAndUser(ctx, entryID, userID)
	return entry, false, err
}

func (r *WaitlistRepo) ClaimNextForNotify(ctx context.Context, slotID uuid.UUID) (*domain.WaitlistEntry, error) {
	const q = `
WITH next_entry AS (
    SELECT id
    FROM waitlist_entries
    WHERE slot_id = $1
      AND status = 'active'
    ORDER BY position
    LIMIT 1
    FOR UPDATE SKIP LOCKED
)
UPDATE waitlist_entries e
SET status = 'notified',
    notified_at = NOW()
FROM next_entry
WHERE e.id = next_entry.id
RETURNING e.id, e.slot_id, e.user_id, e.status, e.position, e.created_at, e.notified_at
`
	db := dbFromContext(ctx, r.pool)
	row := db.QueryRow(ctx, q, slotID)
	entry, err := scanWaitlistEntry(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return entry, nil
}

func (r *WaitlistRepo) getByIDAndUser(ctx context.Context, entryID, userID uuid.UUID) (*domain.WaitlistEntry, error) {
	const q = `
SELECT id, slot_id, user_id, status, position, created_at, notified_at
FROM waitlist_entries
WHERE id = $1
  AND user_id = $2
`
	db := dbFromContext(ctx, r.pool)
	row := db.QueryRow(ctx, q, entryID, userID)
	entry, err := scanWaitlistEntry(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return entry, nil
}

type waitlistScanner interface {
	Scan(dest ...any) error
}

func scanWaitlistEntry(row waitlistScanner) (*domain.WaitlistEntry, error) {
	var entry domain.WaitlistEntry
	if err := row.Scan(
		&entry.ID,
		&entry.SlotID,
		&entry.UserID,
		&entry.Status,
		&entry.Position,
		&entry.CreatedAt,
		&entry.NotifiedAt,
	); err != nil {
		return nil, err
	}
	return &entry, nil
}

func isWaitlistDuplicateJoin(err error) bool {
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) {
		return false
	}
	return pgErr.Code == "23505" && pgErr.ConstraintName == "uq_waitlist_slot_user_active_or_notified"
}
