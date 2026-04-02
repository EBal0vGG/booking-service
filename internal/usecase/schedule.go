package usecase

import (
	"context"

	"booking-service/internal/domain"

	"github.com/google/uuid"
)

type CreateScheduleInput struct {
	DaysOfWeek []int
	StartTime  domain.TimeOfDay
	EndTime    domain.TimeOfDay
}

type ScheduleRule struct {
	DayOfWeek int
	StartTime domain.TimeOfDay
	EndTime   domain.TimeOfDay
}

// ScheduleView represents the "expanded" DB view of a schedule.
// API has a single object with daysOfWeek[], while DB stores one row per (room_id, day_of_week).
type ScheduleView struct {
	RoomID uuid.UUID
	Rules  []ScheduleRule
}

type ScheduleUsecase interface {
	// CreateSchedule must be allowed only for admin.
	// It is allowed only once per room (immutable schedule).
	//
	// daysOfWeek (1..7) must be validated and expanded into multiple DB rows (one per day).
	// start_time/end_time must satisfy:
	// - start < end
	// - optionally: both are aligned to 30-minute slot grid
	// It returns ScheduleView because input.daysOfWeek[] is expanded into multiple DB rows.
	CreateSchedule(ctx context.Context, user domain.User, roomID uuid.UUID, input CreateScheduleInput) (ScheduleView, error)
}

