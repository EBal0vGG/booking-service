package domain

import (
	"time"

	"github.com/google/uuid"
)

type TimeOfDay string

type Schedule struct {
	ID        uuid.UUID
	RoomID    uuid.UUID
	DayOfWeek int
	StartTime TimeOfDay
	EndTime   TimeOfDay
	CreatedAt time.Time
}
