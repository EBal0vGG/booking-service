package usecase

import (
	"context"
	"time"

	"booking-service/internal/domain"

	"github.com/google/uuid"
)

type SlotUsecase interface {
	// ListAvailableSlots returns only free slots:
	// slots that do NOT have an active booking (bookings.status='active') for this slot_id.
	//
	// date must represent the UTC calendar date (YYYY-MM-DD) for the requested window.
	ListAvailableSlots(ctx context.Context, user domain.User, roomID uuid.UUID, date time.Time) ([]domain.Slot, error)
}

