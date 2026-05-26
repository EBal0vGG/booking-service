package domain

import (
	"time"

	"github.com/google/uuid"
)

type Slot struct {
	ID        uuid.UUID
	RoomID    uuid.UUID
	StartTime time.Time
	EndTime   time.Time
	CreatedAt time.Time
}

type SlotStatus string

const (
	SlotStatusAvailable SlotStatus = "available"
	SlotStatusBooked    SlotStatus = "booked"
	SlotStatusReserved  SlotStatus = "reserved"
	SlotStatusPast      SlotStatus = "past"
)

type SlotView struct {
	ID              uuid.UUID
	RoomID          uuid.UUID
	StartTime       time.Time
	EndTime         time.Time
	Status          SlotStatus
	BookingID       *uuid.UUID
	ReservationID   *uuid.UUID
	WaitlistEntryID *uuid.UUID
}
