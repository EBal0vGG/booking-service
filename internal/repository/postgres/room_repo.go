package postgres

import (
	"context"

	"booking-service/internal/domain"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type RoomRepo struct {
	pool *pgxpool.Pool
}

func NewRoomRepo(pool *pgxpool.Pool) *RoomRepo {
	return &RoomRepo{pool: pool}
}

func (r *RoomRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.Room, error) {
	const q = `
SELECT id, name, description, capacity, created_at
FROM rooms
WHERE id = $1
`
	db := dbFromContext(ctx, r.pool)

	var room domain.Room
	var desc pgtype.Text
	var cap pgtype.Int4
	if err := db.QueryRow(ctx, q, id).Scan(&room.ID, &room.Name, &desc, &cap, &room.CreatedAt); err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	if desc.Valid {
		s := desc.String
		room.Description = &s
	}
	if cap.Valid {
		v := int(cap.Int32)
		room.Capacity = &v
	}
	return &room, nil
}

func (r *RoomRepo) Create(ctx context.Context, room domain.Room) error {
	const q = `
INSERT INTO rooms (id, name, description, capacity, created_at)
VALUES ($1, $2, $3, $4, $5)
`
	db := dbFromContext(ctx, r.pool)
	_, err := db.Exec(ctx, q, room.ID, room.Name, room.Description, room.Capacity, room.CreatedAt)
	return err
}

func (r *RoomRepo) List(ctx context.Context) ([]domain.Room, error) {
	const q = `
SELECT id, name, description, capacity, created_at
FROM rooms
ORDER BY created_at DESC
`
	db := dbFromContext(ctx, r.pool)
	rows, err := db.Query(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []domain.Room
	for rows.Next() {
		var room domain.Room
		var desc pgtype.Text
		var cap pgtype.Int4
		if err := rows.Scan(&room.ID, &room.Name, &desc, &cap, &room.CreatedAt); err != nil {
			return nil, err
		}
		if desc.Valid {
			s := desc.String
			room.Description = &s
		}
		if cap.Valid {
			v := int(cap.Int32)
			room.Capacity = &v
		}
		out = append(out, room)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func (r *RoomRepo) ListWithSchedule(ctx context.Context) ([]domain.Room, error) {
	const q = `
SELECT r.id, r.name, r.description, r.capacity, r.created_at
FROM rooms r
WHERE EXISTS (SELECT 1 FROM schedules s WHERE s.room_id = r.id)
ORDER BY r.id
`
	db := dbFromContext(ctx, r.pool)
	rows, err := db.Query(ctx, q)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []domain.Room
	for rows.Next() {
		var room domain.Room
		var desc pgtype.Text
		var cap pgtype.Int4
		if err := rows.Scan(&room.ID, &room.Name, &desc, &cap, &room.CreatedAt); err != nil {
			return nil, err
		}
		if desc.Valid {
			s := desc.String
			room.Description = &s
		}
		if cap.Valid {
			v := int(cap.Int32)
			room.Capacity = &v
		}
		out = append(out, room)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}
