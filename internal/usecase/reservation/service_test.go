package reservation

import (
	"context"
	"testing"
	"time"

	"booking-service/internal/domain"
	"booking-service/internal/repository"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

type mockTx struct{}

func (mockTx) WithinTransaction(ctx context.Context, fn func(context.Context) error) error {
	return fn(ctx)
}

type mockReservationRepo struct {
	createFn              func(ctx context.Context, reservation domain.SlotReservation) (*domain.SlotReservation, error)
	getByIDFn             func(ctx context.Context, id uuid.UUID) (*domain.SlotReservation, error)
	getByIDForUpdateFn    func(ctx context.Context, id uuid.UUID) (*domain.SlotReservation, error)
	listActiveByUserFn    func(ctx context.Context, userID uuid.UUID, now time.Time) ([]domain.SlotReservation, error)
	getActiveBySlotFn     func(ctx context.Context, slotID uuid.UUID) (*domain.SlotReservation, error)
	getActiveBySlotLockFn func(ctx context.Context, slotID uuid.UUID) (*domain.SlotReservation, error)
	setConfirmedFn        func(ctx context.Context, id uuid.UUID, confirmedAt time.Time) (*domain.SlotReservation, error)
	setCancelledFn        func(ctx context.Context, id uuid.UUID) (*domain.SlotReservation, error)
	expireBatchFn         func(ctx context.Context, now time.Time, limit int) ([]domain.SlotReservation, error)
}

func (m *mockReservationRepo) Create(ctx context.Context, reservation domain.SlotReservation) (*domain.SlotReservation, error) {
	if m.createFn != nil {
		return m.createFn(ctx, reservation)
	}
	return &reservation, nil
}

func (m *mockReservationRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.SlotReservation, error) {
	if m.getByIDFn != nil {
		return m.getByIDFn(ctx, id)
	}
	return nil, nil
}

func (m *mockReservationRepo) GetByIDForUpdate(ctx context.Context, id uuid.UUID) (*domain.SlotReservation, error) {
	if m.getByIDForUpdateFn != nil {
		return m.getByIDForUpdateFn(ctx, id)
	}
	return nil, nil
}

func (m *mockReservationRepo) ListActiveByUser(ctx context.Context, userID uuid.UUID, now time.Time) ([]domain.SlotReservation, error) {
	if m.listActiveByUserFn != nil {
		return m.listActiveByUserFn(ctx, userID, now)
	}
	return nil, nil
}

func (m *mockReservationRepo) GetActiveBySlot(ctx context.Context, slotID uuid.UUID) (*domain.SlotReservation, error) {
	if m.getActiveBySlotFn != nil {
		return m.getActiveBySlotFn(ctx, slotID)
	}
	return nil, nil
}

func (m *mockReservationRepo) GetActiveBySlotForUpdate(ctx context.Context, slotID uuid.UUID) (*domain.SlotReservation, error) {
	if m.getActiveBySlotLockFn != nil {
		return m.getActiveBySlotLockFn(ctx, slotID)
	}
	return nil, nil
}

func (m *mockReservationRepo) SetConfirmed(ctx context.Context, id uuid.UUID, confirmedAt time.Time) (*domain.SlotReservation, error) {
	if m.setConfirmedFn != nil {
		return m.setConfirmedFn(ctx, id, confirmedAt)
	}
	return nil, nil
}

func (m *mockReservationRepo) SetCancelled(ctx context.Context, id uuid.UUID) (*domain.SlotReservation, error) {
	if m.setCancelledFn != nil {
		return m.setCancelledFn(ctx, id)
	}
	return nil, nil
}

func (m *mockReservationRepo) ExpireBatch(ctx context.Context, now time.Time, limit int) ([]domain.SlotReservation, error) {
	if m.expireBatchFn != nil {
		return m.expireBatchFn(ctx, now, limit)
	}
	return nil, nil
}

type mockBookingRepo struct {
	createFn              func(ctx context.Context, booking domain.Booking) error
	getActiveBySlotLockFn func(ctx context.Context, slotID uuid.UUID) (*domain.Booking, error)
}

func (m *mockBookingRepo) Create(ctx context.Context, booking domain.Booking) error {
	if m.createFn != nil {
		return m.createFn(ctx, booking)
	}
	return nil
}

func (m *mockBookingRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.Booking, error) {
	return nil, nil
}

func (m *mockBookingRepo) HasActiveBySlot(ctx context.Context, slotID uuid.UUID) (bool, error) {
	return false, nil
}

func (m *mockBookingRepo) GetActiveBySlotForUpdate(ctx context.Context, slotID uuid.UUID) (*domain.Booking, error) {
	if m.getActiveBySlotLockFn != nil {
		return m.getActiveBySlotLockFn(ctx, slotID)
	}
	return nil, nil
}

func (m *mockBookingRepo) SetCancelled(ctx context.Context, id uuid.UUID) (*domain.Booking, error) {
	return nil, nil
}

func (m *mockBookingRepo) List(ctx context.Context, page, pageSize int) ([]domain.Booking, int, error) {
	return nil, 0, nil
}

func (m *mockBookingRepo) ListFutureByUser(ctx context.Context, userID uuid.UUID, now time.Time) ([]domain.Booking, error) {
	return nil, nil
}

type mockSlotRepo struct {
	getByIDFn func(ctx context.Context, slotID uuid.UUID) (*domain.Slot, error)
}

func (m *mockSlotRepo) ListAvailableByRoomAndDate(ctx context.Context, roomID uuid.UUID, date time.Time) ([]domain.Slot, error) {
	return nil, nil
}

func (m *mockSlotRepo) ListAllByRoomAndDate(ctx context.Context, roomID uuid.UUID, date time.Time, now time.Time) ([]domain.SlotView, error) {
	return nil, nil
}

func (m *mockSlotRepo) GetByID(ctx context.Context, slotID uuid.UUID) (*domain.Slot, error) {
	if m.getByIDFn != nil {
		return m.getByIDFn(ctx, slotID)
	}
	return nil, nil
}

type mockWaitlistRepo struct {
	claimFn func(ctx context.Context, slotID uuid.UUID) (*domain.WaitlistEntry, error)
}

func (m *mockWaitlistRepo) Join(ctx context.Context, entry domain.WaitlistEntry) (*domain.WaitlistEntry, error) {
	return nil, nil
}

func (m *mockWaitlistRepo) Leave(ctx context.Context, entryID, userID uuid.UUID) (*domain.WaitlistEntry, bool, error) {
	return nil, false, nil
}

func (m *mockWaitlistRepo) ClaimNextForNotify(ctx context.Context, slotID uuid.UUID) (*domain.WaitlistEntry, error) {
	if m.claimFn != nil {
		return m.claimFn(ctx, slotID)
	}
	return nil, nil
}

type mockEventPublisher struct {
	slotBookedCalls             int
	slotAvailableCalls          int
	slotReservedCalls           int
	slotReservationExpiredCalls int
	waitlistReservedCalls       int
	reservationExpiredCalls     int
}

func (m *mockEventPublisher) SlotBooked(ctx context.Context, roomID, slotID, bookingID uuid.UUID) {
	m.slotBookedCalls++
}

func (m *mockEventPublisher) SlotReleased(ctx context.Context, roomID, slotID, bookingID uuid.UUID) {}

func (m *mockEventPublisher) SlotAvailable(ctx context.Context, roomID, slotID uuid.UUID) {
	m.slotAvailableCalls++
}

func (m *mockEventPublisher) SlotReserved(ctx context.Context, roomID, slotID, reservationID uuid.UUID) {
	m.slotReservedCalls++
}

func (m *mockEventPublisher) SlotReservationExpired(ctx context.Context, roomID, slotID, reservationID uuid.UUID) {
	m.slotReservationExpiredCalls++
}

func (m *mockEventPublisher) WaitlistSlotReserved(ctx context.Context, roomID, slotID, userID, reservationID, waitlistEntryID uuid.UUID, expiresAt time.Time) {
	m.waitlistReservedCalls++
}

func (m *mockEventPublisher) ReservationExpired(ctx context.Context, roomID, slotID, userID, reservationID uuid.UUID) {
	m.reservationExpiredCalls++
}

var _ repository.ReservationRepository = (*mockReservationRepo)(nil)
var _ repository.BookingRepository = (*mockBookingRepo)(nil)
var _ repository.SlotRepository = (*mockSlotRepo)(nil)
var _ repository.WaitlistRepository = (*mockWaitlistRepo)(nil)

func TestListMyActiveReservations_Success(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 3, 25, 12, 0, 0, 0, time.UTC)
	userID := uuid.New()
	reservationID := uuid.New()
	slotID := uuid.New()

	svc := &Service{
		reservations: &mockReservationRepo{
			listActiveByUserFn: func(ctx context.Context, gotUserID uuid.UUID, gotNow time.Time) ([]domain.SlotReservation, error) {
				require.Equal(t, userID, gotUserID)
				require.Equal(t, now, gotNow)
				return []domain.SlotReservation{
					{
						ID:        reservationID,
						SlotID:    slotID,
						UserID:    userID,
						Status:    domain.ReservationStatusActive,
						ExpiresAt: now.Add(time.Minute),
						CreatedAt: now.Add(-time.Minute),
					},
				}, nil
			},
		},
		now: func() time.Time { return now },
	}

	got, err := svc.ListMyActiveReservations(context.Background(), domain.User{ID: userID, Role: domain.RoleUser})
	require.NoError(t, err)
	require.Len(t, got, 1)
	require.Equal(t, reservationID, got[0].ID)
}

func TestListMyActiveReservations_ForbiddenForAdmin(t *testing.T) {
	t.Parallel()

	svc := &Service{
		reservations: &mockReservationRepo{},
		now:          func() time.Time { return time.Now().UTC() },
	}

	_, err := svc.ListMyActiveReservations(context.Background(), domain.User{ID: uuid.New(), Role: domain.RoleAdmin})
	de, ok := domain.AsDomainError(err)
	require.True(t, ok)
	require.Equal(t, domain.ErrorForbidden, de.Code)
}

func TestConfirmReservation_Success(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 3, 25, 12, 0, 0, 0, time.UTC)
	userID := uuid.New()
	slotID := uuid.New()
	reservationID := uuid.New()
	roomID := uuid.New()

	events := &mockEventPublisher{}
	svc := &Service{
		tx: mockTx{},
		reservations: &mockReservationRepo{
			getByIDForUpdateFn: func(ctx context.Context, id uuid.UUID) (*domain.SlotReservation, error) {
				require.Equal(t, reservationID, id)
				return &domain.SlotReservation{
					ID:        reservationID,
					SlotID:    slotID,
					UserID:    userID,
					Status:    domain.ReservationStatusActive,
					ExpiresAt: now.Add(5 * time.Minute),
					CreatedAt: now.Add(-time.Minute),
				}, nil
			},
			setConfirmedFn: func(ctx context.Context, id uuid.UUID, confirmedAt time.Time) (*domain.SlotReservation, error) {
				require.Equal(t, reservationID, id)
				return &domain.SlotReservation{
					ID:          reservationID,
					SlotID:      slotID,
					UserID:      userID,
					Status:      domain.ReservationStatusConfirmed,
					ExpiresAt:   now.Add(5 * time.Minute),
					CreatedAt:   now.Add(-time.Minute),
					ConfirmedAt: &confirmedAt,
				}, nil
			},
		},
		bookings: &mockBookingRepo{
			createFn: func(ctx context.Context, booking domain.Booking) error {
				require.Equal(t, userID, booking.UserID)
				require.Equal(t, slotID, booking.SlotID)
				require.Equal(t, domain.BookingStatusActive, booking.Status)
				return nil
			},
		},
		slots: &mockSlotRepo{
			getByIDFn: func(ctx context.Context, gotSlotID uuid.UUID) (*domain.Slot, error) {
				require.Equal(t, slotID, gotSlotID)
				return &domain.Slot{ID: slotID, RoomID: roomID, StartTime: now.Add(time.Hour), EndTime: now.Add(90 * time.Minute)}, nil
			},
		},
		events:         events,
		reservationTTL: 5 * time.Minute,
		now:            func() time.Time { return now },
	}

	booking, reservation, err := svc.ConfirmReservation(context.Background(), domain.User{ID: userID, Role: domain.RoleUser}, reservationID)
	require.NoError(t, err)
	require.Equal(t, slotID, booking.SlotID)
	require.Equal(t, domain.ReservationStatusConfirmed, reservation.Status)
	require.Equal(t, 1, events.slotBookedCalls)
}

func TestConfirmReservation_Expired(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 3, 25, 12, 0, 0, 0, time.UTC)
	userID := uuid.New()
	reservationID := uuid.New()
	svc := &Service{
		tx: mockTx{},
		reservations: &mockReservationRepo{
			getByIDForUpdateFn: func(ctx context.Context, id uuid.UUID) (*domain.SlotReservation, error) {
				return &domain.SlotReservation{
					ID:        reservationID,
					SlotID:    uuid.New(),
					UserID:    userID,
					Status:    domain.ReservationStatusActive,
					ExpiresAt: now.Add(-time.Second),
					CreatedAt: now.Add(-time.Minute),
				}, nil
			},
		},
		bookings: &mockBookingRepo{},
		slots:    &mockSlotRepo{},
		events:   &mockEventPublisher{},
		now:      func() time.Time { return now },
	}

	_, _, err := svc.ConfirmReservation(context.Background(), domain.User{ID: userID, Role: domain.RoleUser}, reservationID)
	de, ok := domain.AsDomainError(err)
	require.True(t, ok)
	require.Equal(t, domain.ErrorInvalidRequest, de.Code)
}

func TestConfirmReservation_PastSlotRejected(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 3, 25, 12, 0, 0, 0, time.UTC)
	userID := uuid.New()
	reservationID := uuid.New()
	svc := &Service{
		tx: mockTx{},
		reservations: &mockReservationRepo{
			getByIDForUpdateFn: func(ctx context.Context, id uuid.UUID) (*domain.SlotReservation, error) {
				return &domain.SlotReservation{
					ID:        reservationID,
					SlotID:    uuid.New(),
					UserID:    userID,
					Status:    domain.ReservationStatusActive,
					ExpiresAt: now.Add(5 * time.Minute),
					CreatedAt: now,
				}, nil
			},
		},
		bookings: &mockBookingRepo{},
		slots: &mockSlotRepo{
			getByIDFn: func(ctx context.Context, slotID uuid.UUID) (*domain.Slot, error) {
				start := now.Add(-time.Minute)
				return &domain.Slot{ID: slotID, RoomID: uuid.New(), StartTime: start, EndTime: start.Add(30 * time.Minute)}, nil
			},
		},
		events: &mockEventPublisher{},
		now:    func() time.Time { return now },
	}

	_, _, err := svc.ConfirmReservation(context.Background(), domain.User{ID: userID, Role: domain.RoleUser}, reservationID)
	de, ok := domain.AsDomainError(err)
	require.True(t, ok)
	require.Equal(t, domain.ErrorInvalidRequest, de.Code)
}

func TestConfirmReservation_WrongUserForbidden(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 3, 25, 12, 0, 0, 0, time.UTC)
	reservationID := uuid.New()
	svc := &Service{
		tx: mockTx{},
		reservations: &mockReservationRepo{
			getByIDForUpdateFn: func(ctx context.Context, id uuid.UUID) (*domain.SlotReservation, error) {
				return &domain.SlotReservation{
					ID:        reservationID,
					SlotID:    uuid.New(),
					UserID:    uuid.New(),
					Status:    domain.ReservationStatusActive,
					ExpiresAt: now.Add(5 * time.Minute),
					CreatedAt: now,
				}, nil
			},
		},
		bookings: &mockBookingRepo{},
		slots:    &mockSlotRepo{},
		events:   &mockEventPublisher{},
		now:      func() time.Time { return now },
	}

	_, _, err := svc.ConfirmReservation(context.Background(), domain.User{ID: uuid.New(), Role: domain.RoleUser}, reservationID)
	de, ok := domain.AsDomainError(err)
	require.True(t, ok)
	require.Equal(t, domain.ErrorForbidden, de.Code)
}

func TestExpireDue_ExpiredReservationCreatesNextReservation(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 3, 25, 12, 0, 0, 0, time.UTC)
	slotID := uuid.New()
	roomID := uuid.New()
	expiredReservationID := uuid.New()
	nextReservationID := uuid.New()
	waitlistedUser := uuid.New()
	waitlistEntryID := uuid.New()
	events := &mockEventPublisher{}

	svc := &Service{
		tx: mockTx{},
		reservations: &mockReservationRepo{
			expireBatchFn: func(ctx context.Context, at time.Time, limit int) ([]domain.SlotReservation, error) {
				require.Equal(t, now, at)
				require.Equal(t, 10, limit)
				return []domain.SlotReservation{{
					ID:        expiredReservationID,
					SlotID:    slotID,
					UserID:    uuid.New(),
					Status:    domain.ReservationStatusExpired,
					ExpiresAt: now.Add(-time.Minute),
					CreatedAt: now.Add(-10 * time.Minute),
					ExpiredAt: &now,
				}}, nil
			},
			createFn: func(ctx context.Context, reservation domain.SlotReservation) (*domain.SlotReservation, error) {
				require.Equal(t, slotID, reservation.SlotID)
				require.Equal(t, waitlistedUser, reservation.UserID)
				reservation.ID = nextReservationID
				return &reservation, nil
			},
		},
		bookings: &mockBookingRepo{},
		slots: &mockSlotRepo{
			getByIDFn: func(ctx context.Context, gotSlotID uuid.UUID) (*domain.Slot, error) {
				require.Equal(t, slotID, gotSlotID)
				return &domain.Slot{ID: slotID, RoomID: roomID}, nil
			},
		},
		waitlists: &mockWaitlistRepo{
			claimFn: func(ctx context.Context, gotSlotID uuid.UUID) (*domain.WaitlistEntry, error) {
				require.Equal(t, slotID, gotSlotID)
				return &domain.WaitlistEntry{
					ID:       waitlistEntryID,
					SlotID:   slotID,
					UserID:   waitlistedUser,
					Status:   domain.WaitlistStatusNotified,
					Position: 1,
				}, nil
			},
		},
		events:         events,
		reservationTTL: 5 * time.Minute,
		now:            func() time.Time { return now },
	}

	require.NoError(t, svc.ExpireDue(context.Background(), 10))
	require.Equal(t, 1, events.reservationExpiredCalls)
	require.Equal(t, 1, events.slotReservationExpiredCalls)
	require.Equal(t, 1, events.slotReservedCalls)
	require.Equal(t, 1, events.waitlistReservedCalls)
}

func TestCancelReservation_EmptyQueuePublishesSlotAvailable(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 3, 25, 12, 0, 0, 0, time.UTC)
	userID := uuid.New()
	slotID := uuid.New()
	roomID := uuid.New()
	reservationID := uuid.New()
	events := &mockEventPublisher{}
	svc := &Service{
		tx: mockTx{},
		reservations: &mockReservationRepo{
			getByIDForUpdateFn: func(ctx context.Context, id uuid.UUID) (*domain.SlotReservation, error) {
				return &domain.SlotReservation{
					ID:        reservationID,
					SlotID:    slotID,
					UserID:    userID,
					Status:    domain.ReservationStatusActive,
					ExpiresAt: now.Add(5 * time.Minute),
					CreatedAt: now,
				}, nil
			},
			setCancelledFn: func(ctx context.Context, id uuid.UUID) (*domain.SlotReservation, error) {
				return &domain.SlotReservation{
					ID:        reservationID,
					SlotID:    slotID,
					UserID:    userID,
					Status:    domain.ReservationStatusCancelled,
					ExpiresAt: now.Add(5 * time.Minute),
					CreatedAt: now,
				}, nil
			},
		},
		bookings: &mockBookingRepo{},
		slots: &mockSlotRepo{
			getByIDFn: func(ctx context.Context, gotSlotID uuid.UUID) (*domain.Slot, error) {
				require.Equal(t, slotID, gotSlotID)
				return &domain.Slot{ID: slotID, RoomID: roomID}, nil
			},
		},
		waitlists: &mockWaitlistRepo{
			claimFn: func(ctx context.Context, gotSlotID uuid.UUID) (*domain.WaitlistEntry, error) {
				require.Equal(t, slotID, gotSlotID)
				return nil, nil
			},
		},
		events: events,
		now:    func() time.Time { return now },
	}

	_, err := svc.CancelReservation(context.Background(), domain.User{ID: userID, Role: domain.RoleUser}, reservationID)
	require.NoError(t, err)
	require.Equal(t, 1, events.slotAvailableCalls)
	require.Equal(t, 0, events.slotReservedCalls)
	require.Equal(t, 0, events.waitlistReservedCalls)
}

func TestExpireDue_WithoutReplacementPublishesSlotAvailable(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 3, 25, 12, 0, 0, 0, time.UTC)
	slotID := uuid.New()
	roomID := uuid.New()
	expiredReservationID := uuid.New()
	events := &mockEventPublisher{}

	svc := &Service{
		tx: mockTx{},
		reservations: &mockReservationRepo{
			expireBatchFn: func(ctx context.Context, at time.Time, limit int) ([]domain.SlotReservation, error) {
				return []domain.SlotReservation{{
					ID:        expiredReservationID,
					SlotID:    slotID,
					UserID:    uuid.New(),
					Status:    domain.ReservationStatusExpired,
					ExpiresAt: now.Add(-time.Minute),
					CreatedAt: now.Add(-10 * time.Minute),
					ExpiredAt: &now,
				}}, nil
			},
		},
		bookings: &mockBookingRepo{},
		slots: &mockSlotRepo{
			getByIDFn: func(ctx context.Context, gotSlotID uuid.UUID) (*domain.Slot, error) {
				require.Equal(t, slotID, gotSlotID)
				return &domain.Slot{ID: slotID, RoomID: roomID}, nil
			},
		},
		waitlists: &mockWaitlistRepo{
			claimFn: func(ctx context.Context, gotSlotID uuid.UUID) (*domain.WaitlistEntry, error) {
				require.Equal(t, slotID, gotSlotID)
				return nil, nil
			},
		},
		events: events,
		now:    func() time.Time { return now },
	}

	require.NoError(t, svc.ExpireDue(context.Background(), 10))
	require.Equal(t, 1, events.reservationExpiredCalls)
	require.Equal(t, 1, events.slotReservationExpiredCalls)
	require.Equal(t, 1, events.slotAvailableCalls)
	require.Equal(t, 0, events.slotReservedCalls)
}
