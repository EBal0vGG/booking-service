package room

import (
	"context"
	"strings"
	"time"

	"booking-service/internal/domain"
	bookingRepo "booking-service/internal/repository"
	usecase "booking-service/internal/usecase"

	"github.com/google/uuid"
)

type Service struct {
	rooms bookingRepo.RoomRepository
	now   func() time.Time
}

func NewService(rooms bookingRepo.RoomRepository) *Service {
	return &Service{
		rooms: rooms,
		now:   func() time.Time { return time.Now().UTC() },
	}
}

var _ usecase.RoomUsecase = (*Service)(nil)

func (s *Service) ListRooms(ctx context.Context, user domain.User) ([]domain.Room, error) {
	switch user.Role {
	case domain.RoleAdmin, domain.RoleUser:
		return s.rooms.List(ctx)
	default:
		return nil, domain.NewDomainError(domain.ErrorForbidden, "role is not allowed")
	}
}

func (s *Service) CreateRoom(ctx context.Context, user domain.User, input usecase.RoomCreateInput) (domain.Room, error) {
	if user.Role != domain.RoleAdmin {
		return domain.Room{}, domain.NewDomainError(domain.ErrorForbidden, "admin only")
	}
	name := strings.TrimSpace(input.Name)
	if name == "" {
		return domain.Room{}, domain.NewDomainError(domain.ErrorInvalidRequest, "name is required")
	}
	if input.Capacity != nil && *input.Capacity < 0 {
		return domain.Room{}, domain.NewDomainError(domain.ErrorInvalidRequest, "capacity must be >= 0")
	}

	room := domain.Room{
		ID:          uuid.New(),
		Name:        name,
		Description: input.Description,
		Capacity:    input.Capacity,
		CreatedAt:   s.now(),
	}
	if err := s.rooms.Create(ctx, room); err != nil {
		return domain.Room{}, err
	}

	return room, nil
}

