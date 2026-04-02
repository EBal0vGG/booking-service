package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"booking-service/internal/domain"
	authmw "booking-service/internal/transport/http/middleware"
	"booking-service/internal/usecase"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

type mockRoomUC struct {
	listFn   func(ctx context.Context, user domain.User) ([]domain.Room, error)
	createFn func(ctx context.Context, user domain.User, input usecase.RoomCreateInput) (domain.Room, error)
}

func (m *mockRoomUC) ListRooms(ctx context.Context, user domain.User) ([]domain.Room, error) {
	return m.listFn(ctx, user)
}

func (m *mockRoomUC) CreateRoom(ctx context.Context, user domain.User, input usecase.RoomCreateInput) (domain.Room, error) {
	return m.createFn(ctx, user, input)
}

func TestCreateRoom_HappyPath(t *testing.T) {
	t.Parallel()

	secret := "test-secret"
	token := mustJWT(t, secret, domain.RoleAdmin)

	roomID := uuid.New()
	desc := "hello"
	capacity := 12

	uc := &mockRoomUC{
		createFn: func(ctx context.Context, user domain.User, input usecase.RoomCreateInput) (domain.Room, error) {
			require.Equal(t, domain.RoleAdmin, user.Role)
			// Handler passes JSON fields as-is to usecase (normalization happens in usecase).
			require.Equal(t, " room-1 ", input.Name)
			require.Equal(t, &desc, input.Description)
			require.Equal(t, &capacity, input.Capacity)
			return domain.Room{
				ID:          roomID,
				Name:        "room-1",
				Description: &desc,
				Capacity:    &capacity,
				CreatedAt:   time.Date(2026, 3, 25, 12, 0, 0, 0, time.UTC),
			}, nil
		},
	}

	h := NewRoomHandler(uc)
	next := http.HandlerFunc(h.CreateRoom)
	wrapped := authmw.NewAuth(secret).RequireUser(next)

	req := httptest.NewRequest(http.MethodPost, "/rooms/create", strings.NewReader(`{"name":" room-1 ","description":"hello","capacity":12}`))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()
	wrapped.ServeHTTP(rec, req)

	require.Equal(t, http.StatusCreated, rec.Code)

	var body createRoomResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	require.Equal(t, roomID.String(), body.Room.ID)
}

func TestCreateRoom_InvalidJSON_Returns400(t *testing.T) {
	t.Parallel()

	secret := "test-secret"
	token := mustJWT(t, secret, domain.RoleAdmin)

	uc := &mockRoomUC{
		createFn: func(ctx context.Context, user domain.User, input usecase.RoomCreateInput) (domain.Room, error) {
			t.Fatal("should not be called for invalid json")
			return domain.Room{}, nil
		},
	}

	h := NewRoomHandler(uc)
	next := http.HandlerFunc(h.CreateRoom)
	wrapped := authmw.NewAuth(secret).RequireUser(next)

	req := httptest.NewRequest(http.MethodPost, "/rooms/create", strings.NewReader(`{bad json`))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()
	wrapped.ServeHTTP(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
	code := decodeErrorCode(t, rec)
	require.Equal(t, string(domain.ErrorInvalidRequest), code)
}

func TestListRooms_HappyPath(t *testing.T) {
	t.Parallel()

	secret := "test-secret"
	token := mustJWT(t, secret, domain.RoleAdmin)

	room1 := domain.Room{ID: uuid.New(), Name: "r1"}
	room2 := domain.Room{ID: uuid.New(), Name: "r2"}

	uc := &mockRoomUC{
		listFn: func(ctx context.Context, user domain.User) ([]domain.Room, error) {
			require.Equal(t, domain.RoleAdmin, user.Role)
			return []domain.Room{room1, room2}, nil
		},
	}

	h := NewRoomHandler(uc)
	next := http.HandlerFunc(h.ListRooms)
	wrapped := authmw.NewAuth(secret).RequireUser(next)

	req := httptest.NewRequest(http.MethodGet, "/rooms/list", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	rec := httptest.NewRecorder()
	wrapped.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	var out listRoomsResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &out))
	require.Len(t, out.Rooms, 2)
	require.Equal(t, room1.ID.String(), out.Rooms[0].ID)
	require.Equal(t, room2.ID.String(), out.Rooms[1].ID)
}

func TestListRooms_UsecaseNil_Returns500(t *testing.T) {
	t.Parallel()

	secret := "test-secret"
	token := mustJWT(t, secret, domain.RoleAdmin)

	h := NewRoomHandler(nil)
	next := http.HandlerFunc(h.ListRooms)
	wrapped := authmw.NewAuth(secret).RequireUser(next)

	req := httptest.NewRequest(http.MethodGet, "/rooms/list", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	rec := httptest.NewRecorder()
	wrapped.ServeHTTP(rec, req)

	require.Equal(t, http.StatusInternalServerError, rec.Code)
	code := decodeErrorCode(t, rec)
	require.Equal(t, string(domain.ErrorInternalError), code)
}

