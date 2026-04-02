package slotgen

import (
	"context"
	"time"

	"booking-service/internal/domain"

	"github.com/google/uuid"
)

// RoomRepository provides rooms for generation.
// ListWithSchedule returns only rooms that have at least one schedule row — no slots for rooms without schedule (per product rules).
type RoomRepository interface {
	ListWithSchedule(ctx context.Context) ([]domain.Room, error)
}

// ScheduleRepository provides immutable room schedule.
// ListByRoomIDs loads schedules for many rooms in one round-trip (avoids N+1 vs per-room ListByRoomID).
type ScheduleRepository interface {
	ListByRoomIDs(ctx context.Context, roomIDs []uuid.UUID) (map[uuid.UUID][]domain.Schedule, error)
}

// SlotRepository provides slot read/write operations for rolling generation.
// Idempotent inserts via INSERT ... ON CONFLICT DO NOTHING cover duplicates and races; no separate Exists/GetRange required for correctness.
type SlotRepository interface {
	GetLastSlotStartByRoomID(ctx context.Context, roomID uuid.UUID) (*time.Time, error)
	InsertBatchIgnoreConflicts(ctx context.Context, slots []domain.Slot) error
}
