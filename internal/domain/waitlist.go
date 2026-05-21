package domain

import (
	"time"

	"github.com/google/uuid"
)

type WaitlistStatus string

const (
	WaitlistStatusActive    WaitlistStatus = "active"
	WaitlistStatusNotified  WaitlistStatus = "notified"
	WaitlistStatusCancelled WaitlistStatus = "cancelled"
	WaitlistStatusExpired   WaitlistStatus = "expired"
)

type WaitlistEntry struct {
	ID         uuid.UUID
	SlotID     uuid.UUID
	UserID     uuid.UUID
	Status     WaitlistStatus
	Position   int64
	CreatedAt  time.Time
	NotifiedAt *time.Time
}
