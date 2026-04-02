package postgres

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"booking-service/internal/domain"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ScheduleRepo struct {
	pool *pgxpool.Pool
}

func NewScheduleRepo(pool *pgxpool.Pool) *ScheduleRepo {
	return &ScheduleRepo{pool: pool}
}

func (r *ScheduleRepo) ExistsByRoomID(ctx context.Context, roomID uuid.UUID) (bool, error) {
	const q = `
SELECT EXISTS(
    SELECT 1
    FROM schedules
    WHERE room_id = $1
)
`
	db := dbFromContext(ctx, r.pool)
	var exists bool
	if err := db.QueryRow(ctx, q, roomID).Scan(&exists); err != nil {
		return false, err
	}
	return exists, nil
}

func (r *ScheduleRepo) CreateBatch(ctx context.Context, schedules []domain.Schedule) error {
	if len(schedules) == 0 {
		return nil
	}
	var q strings.Builder
	q.WriteString(`
INSERT INTO schedules (id, room_id, day_of_week, start_time, end_time, created_at)
VALUES `)
	args := make([]any, 0, len(schedules)*6)
	for i, s := range schedules {
		if i > 0 {
			q.WriteString(", ")
		}
		base := len(args)
		args = append(args, s.ID, s.RoomID, s.DayOfWeek, string(s.StartTime), string(s.EndTime), s.CreatedAt)
		fmt.Fprintf(&q, "($%d,$%d,$%d,$%d::time,$%d::time,$%d)",
			base+1, base+2, base+3, base+4, base+5, base+6)
	}
	db := dbFromContext(ctx, r.pool)
	if _, err := db.Exec(ctx, q.String(), args...); err != nil {
		if isScheduleUniqueViolation(err) {
			return domain.NewDomainError(domain.ErrorScheduleExists, "schedule already exists")
		}
		return err
	}
	return nil
}

func (r *ScheduleRepo) ListByRoomIDs(ctx context.Context, roomIDs []uuid.UUID) (map[uuid.UUID][]domain.Schedule, error) {
	out := make(map[uuid.UUID][]domain.Schedule, len(roomIDs))
	if len(roomIDs) == 0 {
		return out, nil
	}

	const q = `
SELECT id, room_id, day_of_week, start_time::text, end_time::text, created_at
FROM schedules
WHERE room_id = ANY($1)
ORDER BY room_id, day_of_week, id
`
	db := dbFromContext(ctx, r.pool)
	rows, err := db.Query(ctx, q, roomIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var sch domain.Schedule
		var startStr, endStr string
		if err := rows.Scan(&sch.ID, &sch.RoomID, &sch.DayOfWeek, &startStr, &endStr, &sch.CreatedAt); err != nil {
			return nil, err
		}
		sch.StartTime = domain.TimeOfDay(startStr)
		sch.EndTime = domain.TimeOfDay(endStr)
		out[sch.RoomID] = append(out[sch.RoomID], sch)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func isScheduleUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) {
		return false
	}
	if pgErr.Code != "23505" {
		return false
	}
	// schedules_room_id_key = legacy UNIQUE(room_id); schedules_room_id_day_of_week_key = intended composite unique.
	switch pgErr.ConstraintName {
	case "schedules_room_id_day_of_week_key", "schedules_room_id_key":
		return true
	default:
		return false
	}
}
