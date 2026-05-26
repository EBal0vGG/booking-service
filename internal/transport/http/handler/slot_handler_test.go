package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"booking-service/internal/domain"
	authmw "booking-service/internal/transport/http/middleware"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

type mockSlotUC struct {
	listFn    func(ctx context.Context, user domain.User, roomID uuid.UUID, date time.Time) ([]domain.Slot, error)
	listAllFn func(ctx context.Context, user domain.User, roomID uuid.UUID, date time.Time) ([]domain.SlotView, error)
}

func (m *mockSlotUC) ListAvailableSlots(ctx context.Context, user domain.User, roomID uuid.UUID, date time.Time) ([]domain.Slot, error) {
	return m.listFn(ctx, user, roomID, date)
}

func (m *mockSlotUC) ListRoomSlots(ctx context.Context, user domain.User, roomID uuid.UUID, date time.Time) ([]domain.SlotView, error) {
	if m.listAllFn != nil {
		return m.listAllFn(ctx, user, roomID, date)
	}
	return nil, nil
}

func TestListAvailableSlots_HappyPath(t *testing.T) {
	t.Parallel()

	secret := "test-secret"
	token := mustJWT(t, secret, domain.RoleUser)

	roomID := uuid.New()
	date := time.Date(2026, 3, 25, 0, 0, 0, 0, time.UTC)

	start := time.Date(2026, 3, 25, 10, 0, 0, 0, time.UTC)
	end := start.Add(30 * time.Minute)

	uc := &mockSlotUC{
		listFn: func(ctx context.Context, user domain.User, gotRoomID uuid.UUID, gotDate time.Time) ([]domain.Slot, error) {
			require.Equal(t, roomID, gotRoomID)
			require.Equal(t, date, gotDate)
			require.Equal(t, domain.RoleUser, user.Role)
			return []domain.Slot{
				{ID: uuid.New(), RoomID: roomID, StartTime: start, EndTime: end},
			}, nil
		},
	}

	h := NewSlotHandler(uc)
	next := http.HandlerFunc(h.ListAvailableSlots)
	wrapped := authmw.NewAuth(secret).RequireUser(next)

	req := httptest.NewRequest(http.MethodGet, "/rooms/"+roomID.String()+"/slots/list?date="+date.Format("2006-01-02"), nil)
	req = setChiParam(req, "roomId", roomID.String())
	req.Header.Set("Authorization", "Bearer "+token)

	rec := httptest.NewRecorder()
	wrapped.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	var out listSlotsResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &out))
	require.Len(t, out.Slots, 1)
	require.Equal(t, start.UTC().Format(time.RFC3339), out.Slots[0].StartAt)
	require.Equal(t, end.UTC().Format(time.RFC3339), out.Slots[0].EndAt)
}

func TestListAvailableSlots_MissingDate_Returns400(t *testing.T) {
	t.Parallel()

	secret := "test-secret"
	token := mustJWT(t, secret, domain.RoleUser)

	uc := &mockSlotUC{
		listFn: func(ctx context.Context, user domain.User, roomID uuid.UUID, date time.Time) ([]domain.Slot, error) {
			t.Fatal("should not be called")
			return nil, nil
		},
	}

	h := NewSlotHandler(uc)
	next := http.HandlerFunc(h.ListAvailableSlots)
	wrapped := authmw.NewAuth(secret).RequireUser(next)

	roomID := uuid.New()
	req := httptest.NewRequest(http.MethodGet, "/rooms/"+roomID.String()+"/slots/list", nil)
	req = setChiParam(req, "roomId", roomID.String())
	req.Header.Set("Authorization", "Bearer "+token)

	rec := httptest.NewRecorder()
	wrapped.ServeHTTP(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
	code := decodeErrorCode(t, rec)
	require.Equal(t, string(domain.ErrorInvalidRequest), code)
}

func TestListRoomSlots_HappyPath(t *testing.T) {
	t.Parallel()

	secret := "test-secret"
	token := mustJWT(t, secret, domain.RoleUser)
	roomID := uuid.New()
	date := time.Date(2026, 3, 25, 0, 0, 0, 0, time.UTC)
	slotID := uuid.New()
	bookingID := uuid.New()

	uc := &mockSlotUC{
		listFn: func(ctx context.Context, user domain.User, roomID uuid.UUID, date time.Time) ([]domain.Slot, error) {
			return nil, nil
		},
		listAllFn: func(ctx context.Context, user domain.User, gotRoomID uuid.UUID, gotDate time.Time) ([]domain.SlotView, error) {
			require.Equal(t, roomID, gotRoomID)
			require.Equal(t, date, gotDate)
			return []domain.SlotView{
				{
					ID:        slotID,
					RoomID:    roomID,
					StartTime: date.Add(9 * time.Hour),
					EndTime:   date.Add(9*time.Hour + 30*time.Minute),
					Status:    domain.SlotStatusBooked,
					BookingID: &bookingID,
				},
			}, nil
		},
	}

	h := NewSlotHandler(uc)
	next := http.HandlerFunc(h.ListRoomSlots)
	wrapped := authmw.NewAuth(secret).RequireUser(next)

	req := httptest.NewRequest(http.MethodGet, "/rooms/"+roomID.String()+"/slots/all?date="+date.Format("2006-01-02"), nil)
	req = setChiParam(req, "roomId", roomID.String())
	req.Header.Set("Authorization", "Bearer "+token)

	rec := httptest.NewRecorder()
	wrapped.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	var out listSlotViewsResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &out))
	require.Len(t, out.Slots, 1)
	require.Equal(t, string(domain.SlotStatusBooked), out.Slots[0].Status)
	require.NotNil(t, out.Slots[0].BookingID)
	require.Equal(t, bookingID.String(), *out.Slots[0].BookingID)
}
