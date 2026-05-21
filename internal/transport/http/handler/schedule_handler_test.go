package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"booking-service/internal/domain"
	authmw "booking-service/internal/transport/http/middleware"
	bookingusecase "booking-service/internal/usecase"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

type mockScheduleUC struct {
	createFn func(ctx context.Context, user domain.User, roomID uuid.UUID, input bookingusecase.CreateScheduleInput) (bookingusecase.ScheduleView, error)
}

func (m *mockScheduleUC) CreateSchedule(ctx context.Context, user domain.User, roomID uuid.UUID, input bookingusecase.CreateScheduleInput) (bookingusecase.ScheduleView, error) {
	return m.createFn(ctx, user, roomID, input)
}

func TestCreateSchedule_HappyPath(t *testing.T) {
	t.Parallel()

	secret := "test-secret"
	token := mustJWT(t, secret, domain.RoleAdmin)

	roomID := uuid.New()

	view := bookingusecase.ScheduleView{
		RoomID: roomID,
		Rules: []bookingusecase.ScheduleRule{
			{DayOfWeek: 3, StartTime: domain.TimeOfDay("09:00"), EndTime: domain.TimeOfDay("18:00")},
			{DayOfWeek: 1, StartTime: domain.TimeOfDay("09:00"), EndTime: domain.TimeOfDay("18:00")},
		},
	}

	h := NewScheduleHandler(&mockScheduleUC{
		createFn: func(ctx context.Context, user domain.User, rid uuid.UUID, input bookingusecase.CreateScheduleInput) (bookingusecase.ScheduleView, error) {
			require.Equal(t, roomID, rid)
			require.Equal(t, domain.RoleAdmin, user.Role)
			return view, nil
		},
	})

	req := httptest.NewRequest(http.MethodPost, "/rooms/"+roomID.String()+"/schedule/create", strings.NewReader(`{"daysOfWeek":[3,1],"startTime":"09:00","endTime":"18:00"}`))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	routeCtx := chi.NewRouteContext()
	routeCtx.URLParams.Add("roomId", roomID.String())
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, routeCtx))

	rec := httptest.NewRecorder()
	next := http.HandlerFunc(h.CreateSchedule)
	authmw.NewAuth(secret).RequireUser(next).ServeHTTP(rec, req)

	require.Equal(t, http.StatusCreated, rec.Code)
	var out createScheduleResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &out))
	require.Equal(t, roomID.String(), out.Schedule.RoomID)
	require.Equal(t, []int{1, 3}, out.Schedule.DaysOfWeek)
	require.Equal(t, "09:00", out.Schedule.StartTime)
	require.Equal(t, "18:00", out.Schedule.EndTime)
}

func TestCreateSchedule_InvalidJSON_Returns400(t *testing.T) {
	t.Parallel()

	secret := "test-secret"
	token := mustJWT(t, secret, domain.RoleAdmin)
	roomID := uuid.New()

	h := NewScheduleHandler(&mockScheduleUC{
		createFn: func(ctx context.Context, user domain.User, rid uuid.UUID, input bookingusecase.CreateScheduleInput) (bookingusecase.ScheduleView, error) {
			t.Fatal("should not be called")
			return bookingusecase.ScheduleView{}, nil
		},
	})

	req := httptest.NewRequest(http.MethodPost, "/rooms/"+roomID.String()+"/schedule/create", strings.NewReader(`{bad json`))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	routeCtx := chi.NewRouteContext()
	routeCtx.URLParams.Add("roomId", roomID.String())
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, routeCtx))

	rec := httptest.NewRecorder()
	next := http.HandlerFunc(h.CreateSchedule)
	authmw.NewAuth(secret).RequireUser(next).ServeHTTP(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
	code := decodeErrorCode(t, rec)
	require.Equal(t, string(domain.ErrorInvalidRequest), code)
}
