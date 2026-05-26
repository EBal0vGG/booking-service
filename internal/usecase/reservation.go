package usecase

import (
	"context"

	"booking-service/internal/domain"

	"github.com/google/uuid"
)

type ReservationUsecase interface {
	ConfirmReservation(ctx context.Context, user domain.User, reservationID uuid.UUID) (domain.Booking, domain.SlotReservation, error)
	CancelReservation(ctx context.Context, user domain.User, reservationID uuid.UUID) (domain.SlotReservation, error)
	ListMyActiveReservations(ctx context.Context, user domain.User) ([]domain.SlotReservation, error)
}

type ReservationExpirationUsecase interface {
	ExpireDue(ctx context.Context, limit int) error
}
