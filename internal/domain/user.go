package domain

import (
	"time"

	"github.com/google/uuid"
)

type UserRole string

const (
	RoleAdmin UserRole = "admin"
	RoleUser  UserRole = "user"
)

type User struct {
	ID           uuid.UUID
	Email        string
	PasswordHash *string
	Role         UserRole
	CreatedAt    time.Time
}
