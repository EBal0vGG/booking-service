package usecase

import (
	"context"

	"booking-service/internal/domain"
)

type AuthUsecase interface {
	Register(ctx context.Context, email, password string, role domain.UserRole) (user domain.User, err error)
	Login(ctx context.Context, email, password string) (token string, err error)

	// DummyLogin returns a test JWT token for the given role.
	// - user_id is fixed per role
	// - role is validated to be within allowed values
	DummyLogin(ctx context.Context, role domain.UserRole) (token string, err error)
}
