package usecase

import (
	"context"

	"booking-service/internal/domain"

	"github.com/google/uuid"
)

type BookingUsecase interface {
	// CreateBooking must run in a DB transaction:
	// - validate slot exists
	// - validate slot.start >= now (UTC)
	// - insert booking relying on DB constraints (partial unique index + idempotent insert)
	//   (avoid a separate "check then insert" race; handle unique violation / conflict)
	CreateBooking(ctx context.Context, user domain.User, slotID uuid.UUID, createConferenceLink bool) (domain.Booking, error)

	// ListBookings is admin-only. Supports pagination.
	// ListBookings should return bookings ordered either by:
	// - `created_at DESC` (recommended for admin UI), or
	// - slot start DESC
	ListBookings(ctx context.Context, user domain.User, page, pageSize int) ([]domain.Booking, domain.Pagination, error)

	// ListMyBookings returns only future bookings (slot.start >= now), user-owned only.
	ListMyBookings(ctx context.Context, user domain.User) ([]domain.Booking, error)

	// CancelBooking must be idempotent and user-owned:
	// - user can cancel only his booking
	// - if already cancelled, return current booking state
	// - if booking doesn't exist, return BOOKING_NOT_FOUND
	//
	// Usually implemented inside a DB transaction.
	// CancelBooking is idempotent: repeated cancel returns the current booking state.
	// Must be implemented via SetCancelled(...) in repository (transaction).
	CancelBooking(ctx context.Context, user domain.User, bookingID uuid.UUID) (domain.Booking, error)
}

