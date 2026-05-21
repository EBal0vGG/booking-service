package booking

import (
	"context"
	"testing"
	"time"

	"booking-service/internal/domain"
	"booking-service/internal/repository"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

type mockTx struct {
	run func(ctx context.Context, fn func(context.Context) error) error
}

func (m *mockTx) WithinTransaction(ctx context.Context, fn func(context.Context) error) error {
	if m.run != nil {
		return m.run(ctx, fn)
	}
	return fn(ctx)
}

type mockBookings struct {
	createErr error
	created   domain.Booking

	getByID          *domain.Booking
	cancelled        *domain.Booking
	hasActiveBySlot  bool
	hasActiveSlotErr error
}

func (m *mockBookings) Create(ctx context.Context, booking domain.Booking) error {
	m.created = booking
	return m.createErr
}

func (m *mockBookings) GetByID(ctx context.Context, id uuid.UUID) (*domain.Booking, error) {
	return m.getByID, nil
}

func (m *mockBookings) SetCancelled(ctx context.Context, id uuid.UUID) (*domain.Booking, error) {
	return m.cancelled, nil
}

func (m *mockBookings) HasActiveBySlot(ctx context.Context, slotID uuid.UUID) (bool, error) {
	if m.hasActiveSlotErr != nil {
		return false, m.hasActiveSlotErr
	}
	return m.hasActiveBySlot, nil
}

func (m *mockBookings) GetActiveBySlotForUpdate(ctx context.Context, slotID uuid.UUID) (*domain.Booking, error) {
	if m.hasActiveSlotErr != nil {
		return nil, m.hasActiveSlotErr
	}
	if !m.hasActiveBySlot {
		return nil, nil
	}
	return &domain.Booking{ID: uuid.New(), SlotID: slotID, Status: domain.BookingStatusActive}, nil
}

func (m *mockBookings) List(ctx context.Context, page, pageSize int) ([]domain.Booking, int, error) {
	return nil, 0, nil
}

func (m *mockBookings) ListFutureByUser(ctx context.Context, userID uuid.UUID, now time.Time) ([]domain.Booking, error) {
	return nil, nil
}

var _ repository.BookingRepository = (*mockBookings)(nil)

type mockSlots struct {
	byID *domain.Slot
	err  error
}

func (m *mockSlots) GetByID(ctx context.Context, slotID uuid.UUID) (*domain.Slot, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.byID, nil
}

func (m *mockSlots) ListAvailableByRoomAndDate(ctx context.Context, roomID uuid.UUID, date time.Time) ([]domain.Slot, error) {
	return nil, nil
}

var _ repository.SlotRepository = (*mockSlots)(nil)

type mockWaitlists struct {
	claimFn func(ctx context.Context, slotID uuid.UUID) (*domain.WaitlistEntry, error)
}

func (m *mockWaitlists) Join(ctx context.Context, entry domain.WaitlistEntry) (*domain.WaitlistEntry, error) {
	return nil, nil
}

func (m *mockWaitlists) Leave(ctx context.Context, entryID, userID uuid.UUID) (*domain.WaitlistEntry, bool, error) {
	return nil, false, nil
}

func (m *mockWaitlists) ClaimNextForNotify(ctx context.Context, slotID uuid.UUID) (*domain.WaitlistEntry, error) {
	if m.claimFn != nil {
		return m.claimFn(ctx, slotID)
	}
	return nil, nil
}

var _ repository.WaitlistRepository = (*mockWaitlists)(nil)

type mockEventPublisher struct {
	slotReleasedCalls int
	waitlistCalls     int
	lastWaitlistUser  uuid.UUID
}

func (m *mockEventPublisher) SlotBooked(ctx context.Context, roomID, slotID, bookingID uuid.UUID) {
}

func (m *mockEventPublisher) SlotReleased(ctx context.Context, roomID, slotID, bookingID uuid.UUID) {
	m.slotReleasedCalls++
}

func (m *mockEventPublisher) WaitlistSlotAvailable(ctx context.Context, roomID, slotID, userID, waitlistEntryID uuid.UUID) {
	m.waitlistCalls++
	m.lastWaitlistUser = userID
}

func TestCreateBooking_ForbiddenForAdmin(t *testing.T) {
	t.Parallel()
	svc := &Service{
		tx:       &mockTx{},
		bookings: &mockBookings{},
		slots:    &mockSlots{},
		now:      func() time.Time { return time.Date(2025, 3, 25, 12, 0, 0, 0, time.UTC) },
	}
	admin := domain.User{ID: uuid.New(), Role: domain.RoleAdmin}
	_, err := svc.CreateBooking(context.Background(), admin, uuid.New(), false)
	de, ok := domain.AsDomainError(err)
	require.True(t, ok)
	require.Equal(t, domain.ErrorForbidden, de.Code)
}

func TestCreateBooking_SlotNotFound(t *testing.T) {
	t.Parallel()
	svc := &Service{
		tx:       &mockTx{},
		bookings: &mockBookings{},
		slots:    &mockSlots{byID: nil},
		now:      func() time.Time { return time.Date(2025, 3, 25, 12, 0, 0, 0, time.UTC) },
	}
	u := domain.User{ID: uuid.New(), Role: domain.RoleUser}
	_, err := svc.CreateBooking(context.Background(), u, uuid.New(), false)
	de, ok := domain.AsDomainError(err)
	require.True(t, ok)
	require.Equal(t, domain.ErrorSlotNotFound, de.Code)
}

func TestCreateBooking_PastSlot(t *testing.T) {
	t.Parallel()
	now := time.Date(2025, 3, 25, 12, 0, 0, 0, time.UTC)
	past := now.Add(-time.Hour)
	svc := &Service{
		tx:       &mockTx{},
		bookings: &mockBookings{},
		slots: &mockSlots{
			byID: &domain.Slot{
				ID:        uuid.New(),
				StartTime: past,
				EndTime:   past.Add(30 * time.Minute),
			},
		},
		now: func() time.Time { return now },
	}
	u := domain.User{ID: uuid.New(), Role: domain.RoleUser}
	_, err := svc.CreateBooking(context.Background(), u, uuid.New(), false)
	de, ok := domain.AsDomainError(err)
	require.True(t, ok)
	require.Equal(t, domain.ErrorInvalidRequest, de.Code)
}

func TestCreateBooking_Success(t *testing.T) {
	t.Parallel()
	now := time.Date(2025, 3, 25, 12, 0, 0, 0, time.UTC)
	slotID := uuid.New()
	futureStart := now.Add(time.Hour)
	br := &mockBookings{}
	svc := &Service{
		tx:       &mockTx{},
		bookings: br,
		slots: &mockSlots{
			byID: &domain.Slot{
				ID:        slotID,
				StartTime: futureStart,
				EndTime:   futureStart.Add(30 * time.Minute),
			},
		},
		now: func() time.Time { return now },
	}
	u := domain.User{ID: uuid.New(), Role: domain.RoleUser}
	b, err := svc.CreateBooking(context.Background(), u, slotID, false)
	require.NoError(t, err)
	require.Equal(t, u.ID, b.UserID)
	require.Equal(t, slotID, b.SlotID)
	require.Equal(t, domain.BookingStatusActive, b.Status)
	require.Equal(t, u.ID, br.created.UserID)
}

func TestCreateBooking_SlotAlreadyBookedPassesThrough(t *testing.T) {
	t.Parallel()
	now := time.Date(2025, 3, 25, 12, 0, 0, 0, time.UTC)
	slotID := uuid.New()
	future := now.Add(time.Hour)
	svc := &Service{
		tx: &mockTx{},
		bookings: &mockBookings{
			createErr: domain.NewDomainError(domain.ErrorSlotAlreadyBooked, "taken"),
		},
		slots: &mockSlots{
			byID: &domain.Slot{ID: slotID, StartTime: future, EndTime: future.Add(30 * time.Minute)},
		},
		now: func() time.Time { return now },
	}
	u := domain.User{ID: uuid.New(), Role: domain.RoleUser}
	_, err := svc.CreateBooking(context.Background(), u, slotID, false)
	de, ok := domain.AsDomainError(err)
	require.True(t, ok)
	require.Equal(t, domain.ErrorSlotAlreadyBooked, de.Code)
}

func TestCancelBooking_ForbiddenForAdmin(t *testing.T) {
	t.Parallel()
	svc := &Service{tx: &mockTx{}, bookings: &mockBookings{}, slots: &mockSlots{}}
	admin := domain.User{ID: uuid.New(), Role: domain.RoleAdmin}
	_, err := svc.CancelBooking(context.Background(), admin, uuid.New())
	de, ok := domain.AsDomainError(err)
	require.True(t, ok)
	require.Equal(t, domain.ErrorForbidden, de.Code)
}

func TestListBookings_ForbiddenForUser(t *testing.T) {
	t.Parallel()
	svc := &Service{tx: &mockTx{}, bookings: &mockBookings{}, slots: &mockSlots{}}
	u := domain.User{ID: uuid.New(), Role: domain.RoleUser}
	_, _, err := svc.ListBookings(context.Background(), u, 1, 10)
	de, ok := domain.AsDomainError(err)
	require.True(t, ok)
	require.Equal(t, domain.ErrorForbidden, de.Code)
}

func TestCancelBooking_Success(t *testing.T) {
	t.Parallel()
	bid := uuid.New()
	uid := uuid.New()
	active := &domain.Booking{
		ID:     bid,
		UserID: uid,
		Status: domain.BookingStatusActive,
	}
	cancelled := &domain.Booking{
		ID:     bid,
		UserID: uid,
		Status: domain.BookingStatusCancelled,
	}
	br := &mockBookings{getByID: active, cancelled: cancelled}
	svc := &Service{
		tx:       &mockTx{},
		bookings: br,
		slots:    &mockSlots{},
	}
	u := domain.User{ID: uid, Role: domain.RoleUser}
	after, err := svc.CancelBooking(context.Background(), u, bid)
	require.NoError(t, err)
	require.Equal(t, domain.BookingStatusCancelled, after.Status)
}

func TestCancelBooking_WrongUser(t *testing.T) {
	t.Parallel()
	bid := uuid.New()
	active := &domain.Booking{
		ID:     bid,
		UserID: uuid.New(),
		Status: domain.BookingStatusActive,
	}
	svc := &Service{
		tx:       &mockTx{},
		bookings: &mockBookings{getByID: active},
		slots:    &mockSlots{},
	}
	u := domain.User{ID: uuid.New(), Role: domain.RoleUser}
	_, err := svc.CancelBooking(context.Background(), u, bid)
	de, ok := domain.AsDomainError(err)
	require.True(t, ok)
	require.Equal(t, domain.ErrorForbidden, de.Code)
}

func TestCancelBooking_ClaimsAndPublishesWaitlistNotification(t *testing.T) {
	t.Parallel()

	bookingID := uuid.New()
	slotID := uuid.New()
	userID := uuid.New()
	roomID := uuid.New()
	waitlistedUserID := uuid.New()
	active := &domain.Booking{
		ID:     bookingID,
		UserID: userID,
		SlotID: slotID,
		Status: domain.BookingStatusActive,
	}
	cancelled := &domain.Booking{
		ID:     bookingID,
		UserID: userID,
		SlotID: slotID,
		Status: domain.BookingStatusCancelled,
	}
	publisher := &mockEventPublisher{}
	svc := &Service{
		tx:       &mockTx{},
		bookings: &mockBookings{getByID: active, cancelled: cancelled},
		slots: &mockSlots{
			byID: &domain.Slot{ID: slotID, RoomID: roomID},
		},
		waitlists: &mockWaitlists{
			claimFn: func(ctx context.Context, gotSlotID uuid.UUID) (*domain.WaitlistEntry, error) {
				require.Equal(t, slotID, gotSlotID)
				return &domain.WaitlistEntry{
					ID:       uuid.New(),
					SlotID:   slotID,
					UserID:   waitlistedUserID,
					Status:   domain.WaitlistStatusNotified,
					Position: 1,
				}, nil
			},
		},
		events: publisher,
	}

	_, err := svc.CancelBooking(context.Background(), domain.User{ID: userID, Role: domain.RoleUser}, bookingID)
	require.NoError(t, err)
	require.Equal(t, 1, publisher.slotReleasedCalls)
	require.Equal(t, 1, publisher.waitlistCalls)
	require.Equal(t, waitlistedUserID, publisher.lastWaitlistUser)
}

func TestCancelBooking_AlreadyCancelledSkipsReleaseAndWaitlistClaim(t *testing.T) {
	t.Parallel()

	bookingID := uuid.New()
	slotID := uuid.New()
	userID := uuid.New()
	roomID := uuid.New()
	alreadyCancelled := &domain.Booking{
		ID:     bookingID,
		UserID: userID,
		SlotID: slotID,
		Status: domain.BookingStatusCancelled,
	}
	publisher := &mockEventPublisher{}
	claimCalls := 0
	svc := &Service{
		tx:       &mockTx{},
		bookings: &mockBookings{getByID: alreadyCancelled, cancelled: alreadyCancelled},
		slots: &mockSlots{
			byID: &domain.Slot{ID: slotID, RoomID: roomID},
		},
		waitlists: &mockWaitlists{
			claimFn: func(ctx context.Context, slotID uuid.UUID) (*domain.WaitlistEntry, error) {
				claimCalls++
				return nil, nil
			},
		},
		events: publisher,
	}

	_, err := svc.CancelBooking(context.Background(), domain.User{ID: userID, Role: domain.RoleUser}, bookingID)
	require.NoError(t, err)
	require.Equal(t, 0, claimCalls)
	require.Equal(t, 0, publisher.slotReleasedCalls)
	require.Equal(t, 0, publisher.waitlistCalls)
}
