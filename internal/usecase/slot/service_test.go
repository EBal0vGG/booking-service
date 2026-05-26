package slot

import (
	"context"
	"testing"
	"time"

	"booking-service/internal/domain"
	bookingRepo "booking-service/internal/repository"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

type mockRooms struct {
	byID *domain.Room
	err  error
}

func (m *mockRooms) GetByID(ctx context.Context, id uuid.UUID) (*domain.Room, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.byID, nil
}

func (m *mockRooms) Create(ctx context.Context, room domain.Room) error { return nil }

func (m *mockRooms) List(ctx context.Context) ([]domain.Room, error) { return nil, nil }

var _ bookingRepo.RoomRepository = (*mockRooms)(nil)

type mockSlotsRepo struct {
	slots     []domain.Slot
	slotViews []domain.SlotView
	err       error
}

func (m *mockSlotsRepo) GetByID(ctx context.Context, slotID uuid.UUID) (*domain.Slot, error) {
	return nil, nil
}

func (m *mockSlotsRepo) ListAvailableByRoomAndDate(ctx context.Context, roomID uuid.UUID, date time.Time) ([]domain.Slot, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.slots, nil
}

func (m *mockSlotsRepo) ListAllByRoomAndDate(ctx context.Context, roomID uuid.UUID, date time.Time, now time.Time) ([]domain.SlotView, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.slotViews, nil
}

var _ bookingRepo.SlotRepository = (*mockSlotsRepo)(nil)

func TestListAvailableSlots_Forbidden(t *testing.T) {
	t.Parallel()
	s := NewService(&mockRooms{}, &mockSlotsRepo{})
	u := domain.User{ID: uuid.New(), Role: "unknown"}
	_, err := s.ListAvailableSlots(context.Background(), u, uuid.New(), time.Now())
	de, ok := domain.AsDomainError(err)
	require.True(t, ok)
	require.Equal(t, domain.ErrorForbidden, de.Code)
}

func TestListAvailableSlots_RoomNotFound(t *testing.T) {
	t.Parallel()
	s := NewService(&mockRooms{byID: nil}, &mockSlotsRepo{})
	u := domain.User{ID: uuid.New(), Role: domain.RoleUser}
	_, err := s.ListAvailableSlots(context.Background(), u, uuid.New(), time.Now())
	de, ok := domain.AsDomainError(err)
	require.True(t, ok)
	require.Equal(t, domain.ErrorRoomNotFound, de.Code)
}

func TestListAvailableSlots_Success(t *testing.T) {
	t.Parallel()
	rid := uuid.New()
	s := NewService(
		&mockRooms{byID: &domain.Room{ID: rid, Name: "r", CreatedAt: time.Now().UTC()}},
		&mockSlotsRepo{slots: []domain.Slot{{ID: uuid.New(), RoomID: rid}}},
	)
	u := domain.User{ID: uuid.New(), Role: domain.RoleAdmin}
	out, err := s.ListAvailableSlots(context.Background(), u, rid, time.Date(2025, 3, 25, 0, 0, 0, 0, time.UTC))
	require.NoError(t, err)
	require.Len(t, out, 1)
}

func TestListRoomSlots_Success(t *testing.T) {
	t.Parallel()
	rid := uuid.New()
	s := NewService(
		&mockRooms{byID: &domain.Room{ID: rid, Name: "r", CreatedAt: time.Now().UTC()}},
		&mockSlotsRepo{slotViews: []domain.SlotView{{ID: uuid.New(), RoomID: rid, Status: domain.SlotStatusBooked}}},
	)
	u := domain.User{ID: uuid.New(), Role: domain.RoleUser}
	out, err := s.ListRoomSlots(context.Background(), u, rid, time.Date(2025, 3, 25, 0, 0, 0, 0, time.UTC))
	require.NoError(t, err)
	require.Len(t, out, 1)
	require.Equal(t, domain.SlotStatusBooked, out[0].Status)
}
