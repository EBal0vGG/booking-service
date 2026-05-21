package usecase

import (
	"context"

	"booking-service/internal/domain"

	"github.com/google/uuid"
)

type WaitlistUsecase interface {
	JoinWaitlist(ctx context.Context, user domain.User, slotID uuid.UUID) (domain.WaitlistEntry, error)
	LeaveWaitlist(ctx context.Context, user domain.User, entryID uuid.UUID) (domain.WaitlistEntry, error)
}
