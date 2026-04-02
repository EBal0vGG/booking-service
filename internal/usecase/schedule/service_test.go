package schedule

import (
	"context"
	"testing"
	"time"

	"booking-service/internal/domain"
	bookingRepo "booking-service/internal/repository"
	usecase "booking-service/internal/usecase"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

type mockRoomRepo struct {
	byID *domain.Room
	err  error
}

func (m *mockRoomRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.Room, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.byID, nil
}

func (m *mockRoomRepo) Create(ctx context.Context, room domain.Room) error { return nil }

func (m *mockRoomRepo) List(ctx context.Context) ([]domain.Room, error) { return nil, nil }

var _ bookingRepo.RoomRepository = (*mockRoomRepo)(nil)

type mockScheduleRepo struct {
	exists   bool
	existsErr error
	createErr error
}

func (m *mockScheduleRepo) ExistsByRoomID(ctx context.Context, roomID uuid.UUID) (bool, error) {
	return m.exists, m.existsErr
}

func (m *mockScheduleRepo) CreateBatch(ctx context.Context, schedules []domain.Schedule) error {
	return m.createErr
}

func (m *mockScheduleRepo) ListByRoomIDs(ctx context.Context, roomIDs []uuid.UUID) (map[uuid.UUID][]domain.Schedule, error) {
	return nil, nil
}

var _ bookingRepo.ScheduleRepository = (*mockScheduleRepo)(nil)

func TestCreateSchedule_ForbiddenForUser(t *testing.T) {
	t.Parallel()
	s := NewService(&mockRoomRepo{}, &mockScheduleRepo{})
	u := domain.User{ID: uuid.New(), Role: domain.RoleUser}
	_, err := s.CreateSchedule(context.Background(), u, uuid.New(), usecase.CreateScheduleInput{
		DaysOfWeek: []int{1},
		StartTime:  "09:00",
		EndTime:    "10:00",
	})
	de, ok := domain.AsDomainError(err)
	require.True(t, ok)
	require.Equal(t, domain.ErrorForbidden, de.Code)
}

func TestCreateSchedule_RoomNotFound(t *testing.T) {
	t.Parallel()
	s := NewService(&mockRoomRepo{byID: nil}, &mockScheduleRepo{})
	u := domain.User{ID: uuid.New(), Role: domain.RoleAdmin}
	_, err := s.CreateSchedule(context.Background(), u, uuid.New(), usecase.CreateScheduleInput{
		DaysOfWeek: []int{1},
		StartTime:  "09:00",
		EndTime:    "10:00",
	})
	de, ok := domain.AsDomainError(err)
	require.True(t, ok)
	require.Equal(t, domain.ErrorRoomNotFound, de.Code)
}

func TestCreateSchedule_AlreadyExists(t *testing.T) {
	t.Parallel()
	roomID := uuid.New()
	s := NewService(
		&mockRoomRepo{byID: &domain.Room{ID: roomID, Name: "r", CreatedAt: time.Now().UTC()}},
		&mockScheduleRepo{exists: true},
	)
	u := domain.User{ID: uuid.New(), Role: domain.RoleAdmin}
	_, err := s.CreateSchedule(context.Background(), u, roomID, usecase.CreateScheduleInput{
		DaysOfWeek: []int{1},
		StartTime:  "09:00",
		EndTime:    "10:00",
	})
	de, ok := domain.AsDomainError(err)
	require.True(t, ok)
	require.Equal(t, domain.ErrorScheduleExists, de.Code)
}

func TestCreateSchedule_InvalidDay(t *testing.T) {
	t.Parallel()
	roomID := uuid.New()
	s := NewService(
		&mockRoomRepo{byID: &domain.Room{ID: roomID, Name: "r", CreatedAt: time.Now().UTC()}},
		&mockScheduleRepo{},
	)
	u := domain.User{ID: uuid.New(), Role: domain.RoleAdmin}
	_, err := s.CreateSchedule(context.Background(), u, roomID, usecase.CreateScheduleInput{
		DaysOfWeek: []int{8},
		StartTime:  "09:00",
		EndTime:    "10:00",
	})
	de, ok := domain.AsDomainError(err)
	require.True(t, ok)
	require.Equal(t, domain.ErrorInvalidRequest, de.Code)
}

func TestCreateSchedule_Success(t *testing.T) {
	t.Parallel()
	roomID := uuid.New()
	sch := &mockScheduleRepo{}
	s := NewService(
		&mockRoomRepo{byID: &domain.Room{ID: roomID, Name: "r", CreatedAt: time.Now().UTC()}},
		sch,
	)
	u := domain.User{ID: uuid.New(), Role: domain.RoleAdmin}
	view, err := s.CreateSchedule(context.Background(), u, roomID, usecase.CreateScheduleInput{
		DaysOfWeek: []int{1, 3},
		StartTime:  "09:00",
		EndTime:    "10:00",
	})
	require.NoError(t, err)
	require.Equal(t, roomID, view.RoomID)
	require.Len(t, view.Rules, 2)
	require.Equal(t, 1, view.Rules[0].DayOfWeek)
	require.Equal(t, 3, view.Rules[1].DayOfWeek)
}

func TestCreateSchedule_DomainConflictFromRepo(t *testing.T) {
	t.Parallel()
	roomID := uuid.New()
	s := NewService(
		&mockRoomRepo{byID: &domain.Room{ID: roomID, Name: "r", CreatedAt: time.Now().UTC()}},
		&mockScheduleRepo{createErr: domain.NewDomainError(domain.ErrorScheduleExists, "schedule already exists")},
	)
	u := domain.User{ID: uuid.New(), Role: domain.RoleAdmin}
	_, err := s.CreateSchedule(context.Background(), u, roomID, usecase.CreateScheduleInput{
		DaysOfWeek: []int{2},
		StartTime:  "09:00",
		EndTime:    "10:00",
	})
	de, ok := domain.AsDomainError(err)
	require.True(t, ok)
	require.Equal(t, domain.ErrorScheduleExists, de.Code)
}
