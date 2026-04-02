package domain

import (
	"time"

	"github.com/google/uuid"
)

type BookingStatus string

const (
	BookingStatusActive    BookingStatus = "active"
	BookingStatusCancelled BookingStatus = "cancelled"
)

type Booking struct {
	ID             uuid.UUID
	UserID         uuid.UUID
	SlotID         uuid.UUID
	Status         BookingStatus
	ConferenceLink *string
	CreatedAt      time.Time
}
