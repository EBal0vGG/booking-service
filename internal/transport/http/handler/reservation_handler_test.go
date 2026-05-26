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

type mockReservationUC struct {
	confirmFn func(ctx context.Context, user domain.User, reservationID uuid.UUID) (domain.Booking, domain.SlotReservation, error)
	cancelFn  func(ctx context.Context, user domain.User, reservationID uuid.UUID) (domain.SlotReservation, error)
	listFn    func(ctx context.Context, user domain.User) ([]domain.SlotReservation, error)
}

func (m *mockReservationUC) ConfirmReservation(ctx context.Context, user domain.User, reservationID uuid.UUID) (domain.Booking, domain.SlotReservation, error) {
	return m.confirmFn(ctx, user, reservationID)
}

func (m *mockReservationUC) CancelReservation(ctx context.Context, user domain.User, reservationID uuid.UUID) (domain.SlotReservation, error) {
	return m.cancelFn(ctx, user, reservationID)
}

func (m *mockReservationUC) ListMyActiveReservations(ctx context.Context, user domain.User) ([]domain.SlotReservation, error) {
	if m.listFn != nil {
		return m.listFn(ctx, user)
	}
	return nil, nil
}

func TestConfirmReservation_HappyPath(t *testing.T) {
	t.Parallel()

	secret := "test-secret"
	token := mustJWT(t, secret, domain.RoleUser)
	reservationID := uuid.New()
	slotID := uuid.New()
	userID := uuid.New()
	bookingID := uuid.New()
	now := time.Date(2026, 3, 25, 12, 0, 0, 0, time.UTC)

	uc := &mockReservationUC{
		confirmFn: func(ctx context.Context, user domain.User, gotID uuid.UUID) (domain.Booking, domain.SlotReservation, error) {
			require.Equal(t, reservationID, gotID)
			return domain.Booking{
					ID:        bookingID,
					UserID:    userID,
					SlotID:    slotID,
					Status:    domain.BookingStatusActive,
					CreatedAt: now,
				}, domain.SlotReservation{
					ID:        reservationID,
					SlotID:    slotID,
					UserID:    user.ID,
					Status:    domain.ReservationStatusConfirmed,
					ExpiresAt: now.Add(5 * time.Minute),
					CreatedAt: now.Add(-time.Minute),
				}, nil
		},
		cancelFn: func(ctx context.Context, user domain.User, reservationID uuid.UUID) (domain.SlotReservation, error) {
			return domain.SlotReservation{}, nil
		},
		listFn: func(ctx context.Context, user domain.User) ([]domain.SlotReservation, error) {
			return nil, nil
		},
	}

	h := NewReservationHandler(uc)
	wrapped := authmw.NewAuth(secret).RequireUser(http.HandlerFunc(h.ConfirmReservation))

	req := httptest.NewRequest(http.MethodPost, "/reservations/"+reservationID.String()+"/confirm", nil)
	req = setChiParam(req, "reservationId", reservationID.String())
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	wrapped.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	var out confirmReservationResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &out))
	require.Equal(t, bookingID.String(), out.Booking.ID)
	require.Equal(t, reservationID.String(), out.Reservation.ID)
	require.Equal(t, string(domain.ReservationStatusConfirmed), out.Reservation.Status)
}

func TestCancelReservation_InvalidID_Returns400(t *testing.T) {
	t.Parallel()

	secret := "test-secret"
	token := mustJWT(t, secret, domain.RoleUser)
	uc := &mockReservationUC{
		confirmFn: func(ctx context.Context, user domain.User, reservationID uuid.UUID) (domain.Booking, domain.SlotReservation, error) {
			return domain.Booking{}, domain.SlotReservation{}, nil
		},
		cancelFn: func(ctx context.Context, user domain.User, reservationID uuid.UUID) (domain.SlotReservation, error) {
			t.Fatal("cancel should not be called")
			return domain.SlotReservation{}, nil
		},
		listFn: func(ctx context.Context, user domain.User) ([]domain.SlotReservation, error) {
			return nil, nil
		},
	}

	h := NewReservationHandler(uc)
	wrapped := authmw.NewAuth(secret).RequireUser(http.HandlerFunc(h.CancelReservation))

	req := httptest.NewRequest(http.MethodPost, "/reservations/bad/cancel", nil)
	req = setChiParam(req, "reservationId", "bad-id")
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	wrapped.ServeHTTP(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
	require.Equal(t, string(domain.ErrorInvalidRequest), decodeErrorCode(t, rec))
}

func TestListMyActiveReservations_HappyPath(t *testing.T) {
	t.Parallel()

	secret := "test-secret"
	token := mustJWT(t, secret, domain.RoleUser)
	userID := uuid.New()
	reservationID := uuid.New()
	slotID := uuid.New()
	now := time.Date(2026, 3, 25, 12, 0, 0, 0, time.UTC)

	uc := &mockReservationUC{
		confirmFn: func(ctx context.Context, user domain.User, reservationID uuid.UUID) (domain.Booking, domain.SlotReservation, error) {
			return domain.Booking{}, domain.SlotReservation{}, nil
		},
		cancelFn: func(ctx context.Context, user domain.User, reservationID uuid.UUID) (domain.SlotReservation, error) {
			return domain.SlotReservation{}, nil
		},
		listFn: func(ctx context.Context, user domain.User) ([]domain.SlotReservation, error) {
			return []domain.SlotReservation{
				{
					ID:        reservationID,
					SlotID:    slotID,
					UserID:    userID,
					Status:    domain.ReservationStatusActive,
					ExpiresAt: now.Add(5 * time.Minute),
					CreatedAt: now.Add(-time.Minute),
				},
			}, nil
		},
	}

	h := NewReservationHandler(uc)
	wrapped := authmw.NewAuth(secret).RequireUser(http.HandlerFunc(h.ListMyActiveReservations))

	req := httptest.NewRequest(http.MethodGet, "/reservations/my/active", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	wrapped.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	var out listMyActiveReservationsResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &out))
	require.Len(t, out.Reservations, 1)
	require.Equal(t, reservationID.String(), out.Reservations[0].ID)
	require.Equal(t, slotID.String(), out.Reservations[0].SlotID)
}
