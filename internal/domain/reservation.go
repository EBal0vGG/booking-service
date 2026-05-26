package domain

import (
	"time"

	"github.com/google/uuid"
)

type ReservationStatus string

const (
	ReservationStatusActive    ReservationStatus = "active"
	ReservationStatusConfirmed ReservationStatus = "confirmed"
	ReservationStatusExpired   ReservationStatus = "expired"
	ReservationStatusCancelled ReservationStatus = "cancelled"
)

type SlotReservation struct {
	ID              uuid.UUID
	SlotID          uuid.UUID
	UserID          uuid.UUID
	WaitlistEntryID *uuid.UUID
	Status          ReservationStatus
	ExpiresAt       time.Time
	CreatedAt       time.Time
	ConfirmedAt     *time.Time
	ExpiredAt       *time.Time
}
