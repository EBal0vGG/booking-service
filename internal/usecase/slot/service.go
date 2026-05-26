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
	dateUTC, err := s.validateAndNormalizeRequest(ctx, user, roomID, date)
	if err != nil {
		return nil, err
	}

	return s.slots.ListAvailableByRoomAndDate(ctx, roomID, dateUTC)
}

func (s *Service) ListRoomSlots(ctx context.Context, user domain.User, roomID uuid.UUID, date time.Time) ([]domain.SlotView, error) {
	dateUTC, err := s.validateAndNormalizeRequest(ctx, user, roomID, date)
	if err != nil {
		return nil, err
	}

	return s.slots.ListAllByRoomAndDate(ctx, roomID, dateUTC, time.Now().UTC())
}

func (s *Service) validateAndNormalizeRequest(ctx context.Context, user domain.User, roomID uuid.UUID, date time.Time) (time.Time, error) {
	switch user.Role {
	case domain.RoleAdmin, domain.RoleUser:
	default:
		return time.Time{}, domain.NewDomainError(domain.ErrorForbidden, "role is not allowed")
	}

	room, err := s.rooms.GetByID(ctx, roomID)
	if err != nil {
		return time.Time{}, domain.WrapDomainError(domain.ErrorInternalError, "get room", err)
	}
	if room == nil {
		return time.Time{}, domain.NewDomainError(domain.ErrorRoomNotFound, "room not found")
	}

	utcY, utcM, utcD := date.UTC().Date()
	return time.Date(utcY, utcM, utcD, 0, 0, 0, 0, time.UTC), nil
}
