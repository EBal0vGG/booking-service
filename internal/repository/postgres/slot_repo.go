package postgres

import (
	"context"
	"time"

	"booking-service/internal/domain"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type SlotRepo struct {
	pool *pgxpool.Pool
}

func NewSlotRepo(pool *pgxpool.Pool) *SlotRepo {
	return &SlotRepo{pool: pool}
}

func (r *SlotRepo) GetByID(ctx context.Context, slotID uuid.UUID) (*domain.Slot, error) {
	const q = `
SELECT id, room_id, start_time, end_time, created_at
FROM slots
WHERE id = $1
`
	db := dbFromContext(ctx, r.pool)
	var slot domain.Slot
	if err := db.QueryRow(ctx, q, slotID).Scan(&slot.ID, &slot.RoomID, &slot.StartTime, &slot.EndTime, &slot.CreatedAt); err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &slot, nil
}

func (r *SlotRepo) ListAvailableByRoomAndDate(ctx context.Context, roomID uuid.UUID, date time.Time) ([]domain.Slot, error) {
	const q = `
SELECT s.id, s.room_id, s.start_time, s.end_time, s.created_at
FROM slots s
LEFT JOIN bookings b
  ON b.slot_id = s.id AND b.status = 'active'
WHERE s.room_id = $1
  AND s.start_time >= $2
  AND s.start_time < $2 + INTERVAL '1 day'
  AND b.id IS NULL
ORDER BY s.start_time
`
	db := dbFromContext(ctx, r.pool)
	rows, err := db.Query(ctx, q, roomID, date.UTC())
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []domain.Slot
	for rows.Next() {
		var slot domain.Slot
		if err := rows.Scan(&slot.ID, &slot.RoomID, &slot.StartTime, &slot.EndTime, &slot.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, slot)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func (r *SlotRepo) GetLastSlotStartByRoomID(ctx context.Context, roomID uuid.UUID) (*time.Time, error) {
	const q = `
SELECT start_time
FROM slots
WHERE room_id = $1
ORDER BY start_time DESC
LIMIT 1
`
	db := dbFromContext(ctx, r.pool)
	var ts time.Time
	err := db.QueryRow(ctx, q, roomID).Scan(&ts)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &ts, nil
}

func (r *SlotRepo) InsertBatchIgnoreConflicts(ctx context.Context, slots []domain.Slot) error {
	if len(slots) == 0 {
		return nil
	}
	const q = `
INSERT INTO slots (id, room_id, start_time, end_time, created_at)
VALUES ($1, $2, $3, $4, $5)
ON CONFLICT (room_id, start_time) DO NOTHING
`
	db := dbFromContext(ctx, r.pool)
	batch := &pgx.Batch{}
	for _, s := range slots {
		batch.Queue(q, s.ID, s.RoomID, s.StartTime, s.EndTime, s.CreatedAt)
	}
	br := db.SendBatch(ctx, batch)
	defer br.Close()
	for range slots {
		if _, err := br.Exec(); err != nil {
			return err
		}
	}
	return nil
}
