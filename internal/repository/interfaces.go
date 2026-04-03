package repository

import (
	"context"
	"time"

	"booking-service/internal/domain"

	"github.com/google/uuid"
)

// UserRepository stores users for auth/register/login.
type UserRepository interface {
	GetByEmail(ctx context.Context, email string) (*domain.User, error)
	Create(ctx context.Context, user domain.User) error
}

// RoomRepository is storage boundary for rooms.
type RoomRepository interface {
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Room, error)
	Create(ctx context.Context, room domain.Room) error
	List(ctx context.Context) ([]domain.Room, error)
}

// ScheduleRepository stores immutable schedule rows.
type ScheduleRepository interface {
	ExistsByRoomID(ctx context.Context, roomID uuid.UUID) (bool, error)
	CreateBatch(ctx context.Context, schedules []domain.Schedule) error
	ListByRoomIDs(ctx context.Context, roomIDs []uuid.UUID) (map[uuid.UUID][]domain.Schedule, error)
}

// SlotRepository provides read access to generated slots and available-slot queries.
type SlotRepository interface {
	ListAvailableByRoomAndDate(ctx context.Context, roomID uuid.UUID, date time.Time) ([]domain.Slot, error)
	GetByID(ctx context.Context, slotID uuid.UUID) (*domain.Slot, error)
}

// BookingRepository provides booking CRUD and availability checks.
type BookingRepository interface {
	Create(ctx context.Context, booking domain.Booking) error
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Booking, error)
	// SetCancelled is idempotent:
	// - if booking is active -> mark cancelled and return updated row
	// - if booking is already cancelled -> return current row unchanged
	SetCancelled(ctx context.Context, id uuid.UUID) (*domain.Booking, error)
	// List returns bookings and total count (for pagination).
	// Order: created_at DESC (or equivalent documented ordering).
	List(ctx context.Context, page, pageSize int) ([]domain.Booking, int, error)
	// ListFutureByUser returns only future bookings (slot.start >= now), owned by user.
	ListFutureByUser(ctx context.Context, userID uuid.UUID, now time.Time) ([]domain.Booking, error)
}
