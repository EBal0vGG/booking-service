package room

import (
	"context"
	"testing"

	"booking-service/internal/domain"
	bookingRepo "booking-service/internal/repository"
	usecase "booking-service/internal/usecase"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

type mockRoomRepo struct {
	listFn   func(ctx context.Context) ([]domain.Room, error)
	createFn func(ctx context.Context, room domain.Room) error

	listCalled   bool
	createCalled bool
	createdRoom  domain.Room
}

func (m *mockRoomRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.Room, error) {
	return nil, nil
}

func (m *mockRoomRepo) Create(ctx context.Context, room domain.Room) error {
	m.createCalled = true
	m.createdRoom = room
	if m.createFn != nil {
		return m.createFn(ctx, room)
	}
	return nil
}

func (m *mockRoomRepo) List(ctx context.Context) ([]domain.Room, error) {
	m.listCalled = true
	if m.listFn != nil {
		return m.listFn(ctx)
	}
	return nil, nil
}

var _ bookingRepo.RoomRepository = (*mockRoomRepo)(nil)

func TestListRooms_Admin(t *testing.T) {
	t.Parallel()

	repo := &mockRoomRepo{
		listFn: func(ctx context.Context) ([]domain.Room, error) {
			return []domain.Room{{ID: uuid.New(), Name: "r"}}, nil
		},
	}

	svc := NewService(repo)
	admin := domain.User{ID: uuid.New(), Role: domain.RoleAdmin}

	out, err := svc.ListRooms(context.Background(), admin)
	require.NoError(t, err)
	require.Len(t, out, 1)
	require.True(t, repo.listCalled)
}

func TestListRooms_User(t *testing.T) {
	t.Parallel()

	repo := &mockRoomRepo{
		listFn: func(ctx context.Context) ([]domain.Room, error) {
			return []domain.Room{{ID: uuid.New(), Name: "r"}}, nil
		},
	}

	svc := NewService(repo)
	u := domain.User{ID: uuid.New(), Role: domain.RoleUser}

	out, err := svc.ListRooms(context.Background(), u)
	require.NoError(t, err)
	require.Len(t, out, 1)
	require.True(t, repo.listCalled)
}

func TestListRooms_InvalidRole_Forbidden(t *testing.T) {
	t.Parallel()

	repo := &mockRoomRepo{}
	svc := NewService(repo)

	u := domain.User{ID: uuid.New(), Role: domain.UserRole("guest")}
	_, err := svc.ListRooms(context.Background(), u)

	de, ok := domain.AsDomainError(err)
	require.True(t, ok)
	require.Equal(t, domain.ErrorForbidden, de.Code)
	require.False(t, repo.listCalled)
}

func TestCreateRoom_Admin_Success(t *testing.T) {
	t.Parallel()

	repo := &mockRoomRepo{}
	svc := NewService(repo)

	admin := domain.User{ID: uuid.New(), Role: domain.RoleAdmin}
	desc := "d"
	capacity := 10

	room, err := svc.CreateRoom(context.Background(), admin, usecase.RoomCreateInput{
		Name:        "  my-room  ",
		Description: &desc,
		Capacity:    &capacity,
	})
	require.NoError(t, err)
	require.True(t, repo.createCalled)
	require.Equal(t, "my-room", room.Name)
	require.Equal(t, &desc, room.Description)
	require.Equal(t, &capacity, room.Capacity)
}

func TestCreateRoom_ForbiddenForUser(t *testing.T) {
	t.Parallel()

	repo := &mockRoomRepo{}
	svc := NewService(repo)

	u := domain.User{ID: uuid.New(), Role: domain.RoleUser}
	_, err := svc.CreateRoom(context.Background(), u, usecase.RoomCreateInput{Name: "x"})

	de, ok := domain.AsDomainError(err)
	require.True(t, ok)
	require.Equal(t, domain.ErrorForbidden, de.Code)
	require.False(t, repo.createCalled)
}

func TestCreateRoom_InvalidRequest_EmptyName(t *testing.T) {
	t.Parallel()

	repo := &mockRoomRepo{}
	svc := NewService(repo)

	admin := domain.User{ID: uuid.New(), Role: domain.RoleAdmin}
	_, err := svc.CreateRoom(context.Background(), admin, usecase.RoomCreateInput{Name: "   "})

	de, ok := domain.AsDomainError(err)
	require.True(t, ok)
	require.Equal(t, domain.ErrorInvalidRequest, de.Code)
	require.False(t, repo.createCalled)
}

func TestCreateRoom_InvalidRequest_NegativeCapacity(t *testing.T) {
	t.Parallel()

	repo := &mockRoomRepo{}
	svc := NewService(repo)

	admin := domain.User{ID: uuid.New(), Role: domain.RoleAdmin}
	neg := -1
	_, err := svc.CreateRoom(context.Background(), admin, usecase.RoomCreateInput{Name: "x", Capacity: &neg})

	de, ok := domain.AsDomainError(err)
	require.True(t, ok)
	require.Equal(t, domain.ErrorInvalidRequest, de.Code)
	require.False(t, repo.createCalled)
}
