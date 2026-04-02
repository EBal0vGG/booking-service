package slot

import (
	"context"
	"time"

	"booking-service/internal/domain"
	bookingRepo "booking-service/internal/repository"
	usecase "booking-service/internal/usecase"

	"github.com/google/uuid"
)

type Service struct {
	rooms bookingRepo.RoomRepository
	slots bookingRepo.SlotRepository
}

func NewService(rooms bookingRepo.RoomRepository, slots bookingRepo.SlotRepository) *Service {
	return &Service{rooms: rooms, slots: slots}
}

var _ usecase.SlotUsecase = (*Service)(nil)

func (s *Service) ListAvailableSlots(ctx context.Context, user domain.User, roomID uuid.UUID, date time.Time) ([]domain.Slot, error) {
	switch user.Role {
	case domain.RoleAdmin, domain.RoleUser:
	default:
		return nil, domain.NewDomainError(domain.ErrorForbidden, "role is not allowed")
	}

	room, err := s.rooms.GetByID(ctx, roomID)
	if err != nil {
		return nil, domain.WrapDomainError(domain.ErrorInternalError, "get room", err)
	}
	if room == nil {
		return nil, domain.NewDomainError(domain.ErrorRoomNotFound, "room not found")
	}

	utcY, utcM, utcD := date.UTC().Date()
	dateUTC := time.Date(utcY, utcM, utcD, 0, 0, 0, 0, time.UTC)

	return s.slots.ListAvailableByRoomAndDate(ctx, roomID, dateUTC)
}

