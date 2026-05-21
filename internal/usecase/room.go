package usecase

import (
	"context"

	"booking-service/internal/domain"
)

type RoomCreateInput struct {
	Name        string
	Description *string
	Capacity    *int
}

type RoomUsecase interface {
	ListRooms(ctx context.Context, user domain.User) ([]domain.Room, error)
	// CreateRoom must be allowed only for admin role.
	CreateRoom(ctx context.Context, user domain.User, input RoomCreateInput) (domain.Room, error)
	// Optionally useful for further stages.
	// GetRoom(ctx context.Context, id uuid.UUID) (*domain.Room, error)
}
