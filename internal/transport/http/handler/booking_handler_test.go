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

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

type mockBookingUC struct {
	createFn func(ctx context.Context, user domain.User, slotID uuid.UUID, createConferenceLink bool) (domain.Booking, error)
	cancelFn func(ctx context.Context, user domain.User, bookingID uuid.UUID) (domain.Booking, error)
}

func (m *mockBookingUC) CreateBooking(ctx context.Context, user domain.User, slotID uuid.UUID, createConferenceLink bool) (domain.Booking, error) {
	return m.createFn(ctx, user, slotID, createConferenceLink)
}

func (m *mockBookingUC) CancelBooking(ctx context.Context, user domain.User, bookingID uuid.UUID) (domain.Booking, error) {
	return m.cancelFn(ctx, user, bookingID)
}

func (m *mockBookingUC) ListBookings(ctx context.Context, user domain.User, page, pageSize int) ([]domain.Booking, domain.Pagination, error) {
	return nil, domain.Pagination{}, nil
}

func (m *mockBookingUC) ListMyBookings(ctx context.Context, user domain.User) ([]domain.Booking, error) {
	return nil, nil
}

func TestCreateBooking_HappyPath(t *testing.T) {
	t.Parallel()

	secret := "test-secret"
	token := mustJWT(t, secret, domain.RoleUser)

	slotID := uuid.New()
	bookingID := uuid.New()
	now := time.Date(2026, 3, 25, 12, 0, 0, 0, time.UTC)

	uc := &mockBookingUC{
		createFn: func(ctx context.Context, user domain.User, gotSlotID uuid.UUID, createConferenceLink bool) (domain.Booking, error) {
			require.Equal(t, domain.RoleUser, user.Role)
			require.Equal(t, slotID, gotSlotID)
			require.False(t, createConferenceLink)
			return domain.Booking{
				ID:        bookingID,
				UserID:    user.ID,
				SlotID:    slotID,
				Status:    domain.BookingStatusActive,
				CreatedAt: now,
			}, nil
		},
	}

	h := NewBookingHandler(uc)
	next := http.HandlerFunc(h.CreateBooking)
	wrapped := authmw.NewAuth(secret).RequireUser(next)

	req := httptest.NewRequest(http.MethodPost, "/bookings/create", strings.NewReader(`{"slotId":"`+slotID.String()+`","createConferenceLink":false}`))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()
	wrapped.ServeHTTP(rec, req)

	require.Equal(t, http.StatusCreated, rec.Code)
	var out createBookingResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &out))
	require.Equal(t, bookingID.String(), out.Booking.ID)
	require.Equal(t, slotID.String(), out.Booking.SlotID)
	require.Equal(t, domain.BookingStatusActive, domain.BookingStatus(out.Booking.Status))
	require.NotEmpty(t, out.Booking.CreatedAt)
}

func TestCreateBooking_InvalidSlotID_Returns400(t *testing.T) {
	t.Parallel()

	secret := "test-secret"
	token := mustJWT(t, secret, domain.RoleUser)

	uc := &mockBookingUC{
		createFn: func(ctx context.Context, user domain.User, slotID uuid.UUID, createConferenceLink bool) (domain.Booking, error) {
			t.Fatal("should not be called")
			return domain.Booking{}, nil
		},
	}

	h := NewBookingHandler(uc)
	next := http.HandlerFunc(h.CreateBooking)
	wrapped := authmw.NewAuth(secret).RequireUser(next)

	req := httptest.NewRequest(http.MethodPost, "/bookings/create", strings.NewReader(`{"slotId":"bad-uuid","createConferenceLink":false}`))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()
	wrapped.ServeHTTP(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
	code := decodeErrorCode(t, rec)
	require.Equal(t, string(domain.ErrorInvalidRequest), code)
}

func TestCancelBooking_HappyPath(t *testing.T) {
	t.Parallel()

	secret := "test-secret"
	token := mustJWT(t, secret, domain.RoleUser)

	bookingID := uuid.New()
	now := time.Date(2026, 3, 25, 12, 0, 0, 0, time.UTC)

	uc := &mockBookingUC{
		cancelFn: func(ctx context.Context, user domain.User, gotID uuid.UUID) (domain.Booking, error) {
			require.Equal(t, bookingID, gotID)
			return domain.Booking{
				ID:        bookingID,
				UserID:    user.ID,
				SlotID:    uuid.New(),
				Status:    domain.BookingStatusCancelled,
				CreatedAt: now,
			}, nil
		},
	}

	h := NewBookingHandler(uc)
	next := http.HandlerFunc(h.CancelBooking)
	wrapped := authmw.NewAuth(secret).RequireUser(next)

	req := httptest.NewRequest(http.MethodPost, "/bookings/"+bookingID.String()+"/cancel", nil)
	req = setChiParam(req, "bookingId", bookingID.String())

	req.Header.Set("Authorization", "Bearer "+token)

	rec := httptest.NewRecorder()
	wrapped.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	var out cancelBookingResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &out))
	require.Equal(t, bookingID.String(), out.Booking.ID)
}
