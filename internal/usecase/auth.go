package usecase

import (
	"context"

	"booking-service/internal/domain"
)

type AuthUsecase interface {
	// DummyLogin returns a test JWT token for the given role.
	// - user_id is fixed per role
	// - role is validated to be within allowed values
	DummyLogin(ctx context.Context, role domain.UserRole) (token string, err error)
}

