package schedule

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"booking-service/internal/domain"
	bookingRepo "booking-service/internal/repository"
	usecase "booking-service/internal/usecase"

	"github.com/google/uuid"
)

type Service struct {
	rooms     bookingRepo.RoomRepository
	schedules bookingRepo.ScheduleRepository
}

func NewService(rooms bookingRepo.RoomRepository, schedules bookingRepo.ScheduleRepository) *Service {
	return &Service{rooms: rooms, schedules: schedules}
}

var _ usecase.ScheduleUsecase = (*Service)(nil)

func (s *Service) CreateSchedule(
	ctx context.Context,
	user domain.User,
	roomID uuid.UUID,
	input usecase.CreateScheduleInput,
) (usecase.ScheduleView, error) {
	if user.Role != domain.RoleAdmin {
		return usecase.ScheduleView{}, domain.NewDomainError(domain.ErrorForbidden, "admin only")
	}

	if len(input.DaysOfWeek) == 0 {
		return usecase.ScheduleView{}, domain.NewDomainError(domain.ErrorInvalidRequest, "daysOfWeek is required")
	}

	// Validate room exists.
	room, err := s.rooms.GetByID(ctx, roomID)
	if err != nil {
		return usecase.ScheduleView{}, domain.WrapDomainError(domain.ErrorInternalError, "get room", err)
	}
	if room == nil {
		return usecase.ScheduleView{}, domain.NewDomainError(domain.ErrorRoomNotFound, "room not found")
	}

	// Schedule is immutable and can be created only once per room.
	exists, err := s.schedules.ExistsByRoomID(ctx, roomID)
	if err != nil {
		return usecase.ScheduleView{}, domain.WrapDomainError(domain.ErrorInternalError, "check schedule exists", err)
	}
	if exists {
		return usecase.ScheduleView{}, domain.NewDomainError(domain.ErrorScheduleExists, "schedule already exists")
	}

	// Validate and expand days.
	uniqDays := make(map[int]struct{}, len(input.DaysOfWeek))
	for _, d := range input.DaysOfWeek {
		if d < 1 || d > 7 {
			return usecase.ScheduleView{}, domain.NewDomainError(domain.ErrorInvalidRequest, "day_of_week must be 1..7")
		}
		uniqDays[d] = struct{}{}
	}
	days := make([]int, 0, len(uniqDays))
	for d := range uniqDays {
		days = append(days, d)
	}
	sort.Ints(days)

	startDur, err := parseTimeOfDay(input.StartTime)
	if err != nil {
		return usecase.ScheduleView{}, domain.WrapDomainError(domain.ErrorInvalidRequest, "invalid start_time", err)
	}
	endDur, err := parseTimeOfDay(input.EndTime)
	if err != nil {
		return usecase.ScheduleView{}, domain.WrapDomainError(domain.ErrorInvalidRequest, "invalid end_time", err)
	}
	if !isAlignedToSlotGrid(startDur) || !isAlignedToSlotGrid(endDur) {
		return usecase.ScheduleView{}, domain.NewDomainError(domain.ErrorInvalidRequest, "start/end must be aligned to 30 minutes")
	}
	if endDur <= startDur {
		return usecase.ScheduleView{}, domain.NewDomainError(domain.ErrorInvalidRequest, "start_time must be < end_time")
	}

	createdAt := time.Now().UTC()
	rules := make([]usecase.ScheduleRule, 0, len(days))
	dbSchedules := make([]domain.Schedule, 0, len(days))
	for _, day := range days {
		rules = append(rules, usecase.ScheduleRule{
			DayOfWeek: day,
			StartTime:  input.StartTime,
			EndTime:    input.EndTime,
		})
		dbSchedules = append(dbSchedules, domain.Schedule{
			ID:        uuid.New(),
			RoomID:    roomID,
			DayOfWeek: day,
			StartTime: input.StartTime,
			EndTime:   input.EndTime,
			CreatedAt: createdAt,
		})
	}

	if err := s.schedules.CreateBatch(ctx, dbSchedules); err != nil {
		// DB constraints are the ultimate source of truth for schedule conflicts.
		if isScheduleConflictError(err) {
			return usecase.ScheduleView{}, domain.NewDomainError(domain.ErrorScheduleExists, "schedule already exists")
		}
		return usecase.ScheduleView{}, domain.WrapDomainError(domain.ErrorInternalError, "create schedule", err)
	}

	return usecase.ScheduleView{
		RoomID: roomID,
		Rules:  rules,
	}, nil
}

func parseTimeOfDay(tod domain.TimeOfDay) (time.Duration, error) {
	s := strings.TrimSpace(string(tod))
	if s == "" {
		return 0, errors.New("empty time of day")
	}
	for _, layout := range []string{"15:04:05", "15:04"} {
		parsed, err := time.ParseInLocation(layout, s, time.UTC)
		if err == nil {
			// seconds must be 0 if provided (or layout is HH:MM).
			if parsed.Second() != 0 || parsed.Nanosecond() != 0 {
				return 0, errors.New("time must not contain seconds")
			}
			h, m, _ := parsed.Clock()
			return time.Duration(h)*time.Hour + time.Duration(m)*time.Minute, nil
		}
	}
	return 0, fmt.Errorf("failed to parse time of day: %s", s)
}

func isAlignedToSlotGrid(d time.Duration) bool {
	// must be full minute boundary at 30 min grid: ..., 00, 30, 60...
	if d < 0 {
		return false
	}
	// total minutes
	mins := d / time.Minute
	return mins%30 == 0 && d%time.Minute == 0
}

func isScheduleConflictError(err error) bool {
	if de, ok := domain.AsDomainError(err); ok && de.Code == domain.ErrorScheduleExists {
		return true
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "duplicate key") ||
		strings.Contains(msg, "unique constraint") ||
		strings.Contains(msg, "schedules_room_id_day_of_week_key")
}

