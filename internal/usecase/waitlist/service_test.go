package waitlist

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

type mockWaitlistRepo struct {
	joinFn  func(ctx context.Context, entry domain.WaitlistEntry) (*domain.WaitlistEntry, error)
	leaveFn func(ctx context.Context, entryID, userID uuid.UUID) (*domain.WaitlistEntry, bool, error)
}

func (m mockWaitlistRepo) Join(ctx context.Context, entry domain.WaitlistEntry) (*domain.WaitlistEntry, error) {
	if m.joinFn != nil {
		return m.joinFn(ctx, entry)
	}
	return &entry, nil
}

func (m mockWaitlistRepo) Leave(ctx context.Context, entryID, userID uuid.UUID) (*domain.WaitlistEntry, bool, error) {
	if m.leaveFn != nil {
		return m.leaveFn(ctx, entryID, userID)
	}
	return nil, false, nil
}

func (mockWaitlistRepo) ClaimNextForNotify(ctx context.Context, slotID uuid.UUID) (*domain.WaitlistEntry, error) {
	return nil, nil
}

type mockSlotRepo struct {
	slot *domain.Slot
	err  error
}

func (m mockSlotRepo) ListAvailableByRoomAndDate(ctx context.Context, roomID uuid.UUID, date time.Time) ([]domain.Slot, error) {
	return nil, nil
}

func (m mockSlotRepo) ListAllByRoomAndDate(ctx context.Context, roomID uuid.UUID, date time.Time, now time.Time) ([]domain.SlotView, error) {
	return nil, nil
}

func (m mockSlotRepo) GetByID(ctx context.Context, slotID uuid.UUID) (*domain.Slot, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.slot, nil
}

type mockBookingRepo struct {
	active *domain.Booking
	err    error
}

func (mockBookingRepo) Create(ctx context.Context, booking domain.Booking) error { return nil }
func (mockBookingRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.Booking, error) {
	return nil, nil
}
func (m mockBookingRepo) HasActiveBySlot(ctx context.Context, slotID uuid.UUID) (bool, error) {
	if m.err != nil {
		return false, m.err
	}
	return m.active != nil, nil
}
func (m mockBookingRepo) GetActiveBySlotForUpdate(ctx context.Context, slotID uuid.UUID) (*domain.Booking, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.active, nil
}
func (mockBookingRepo) SetCancelled(ctx context.Context, id uuid.UUID) (*domain.Booking, error) {
	return nil, nil
}
func (mockBookingRepo) List(ctx context.Context, page, pageSize int) ([]domain.Booking, int, error) {
	return nil, 0, nil
}
func (mockBookingRepo) ListFutureByUser(ctx context.Context, userID uuid.UUID, now time.Time) ([]domain.Booking, error) {
	return nil, nil
}

var _ repository.WaitlistRepository = (*mockWaitlistRepo)(nil)
var _ repository.BookingRepository = (*mockBookingRepo)(nil)
var _ repository.SlotRepository = (*mockSlotRepo)(nil)

func TestJoinWaitlist_SlotNotBooked(t *testing.T) {
	t.Parallel()

	svc := &Service{
		tx:        mockTx{},
		waitlists: mockWaitlistRepo{},
		slots: mockSlotRepo{
			slot: &domain.Slot{ID: uuid.New(), StartTime: time.Date(2026, 3, 25, 13, 0, 0, 0, time.UTC)},
		},
		bookings: mockBookingRepo{active: nil},
		now:      func() time.Time { return time.Date(2026, 3, 25, 12, 0, 0, 0, time.UTC) },
	}

	_, err := svc.JoinWaitlist(context.Background(), domain.User{ID: uuid.New(), Role: domain.RoleUser}, uuid.New())
	de, ok := domain.AsDomainError(err)
	require.True(t, ok)
	require.Equal(t, domain.ErrorSlotNotBooked, de.Code)
}

func TestJoinWaitlist_DuplicateConflict(t *testing.T) {
	t.Parallel()

	slotID := uuid.New()
	svc := &Service{
		tx: mockTx{},
		waitlists: mockWaitlistRepo{
			joinFn: func(ctx context.Context, entry domain.WaitlistEntry) (*domain.WaitlistEntry, error) {
				require.Equal(t, slotID, entry.SlotID)
				return nil, domain.NewDomainError(domain.ErrorWaitlistJoined, "duplicate")
			},
		},
		slots: mockSlotRepo{
			slot: &domain.Slot{ID: slotID, StartTime: time.Date(2026, 3, 25, 13, 0, 0, 0, time.UTC)},
		},
		bookings: mockBookingRepo{
			active: &domain.Booking{ID: uuid.New(), SlotID: slotID, UserID: uuid.New(), Status: domain.BookingStatusActive},
		},
		now: func() time.Time { return time.Date(2026, 3, 25, 12, 0, 0, 0, time.UTC) },
	}

	_, err := svc.JoinWaitlist(context.Background(), domain.User{ID: uuid.New(), Role: domain.RoleUser}, slotID)
	de, ok := domain.AsDomainError(err)
	require.True(t, ok)
	require.Equal(t, domain.ErrorWaitlistJoined, de.Code)
}

func TestJoinWaitlist_PastSlotRejected(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 3, 25, 12, 0, 0, 0, time.UTC)
	slotID := uuid.New()
	userID := uuid.New()
	svc := &Service{
		tx:        mockTx{},
		waitlists: mockWaitlistRepo{},
		slots: mockSlotRepo{
			slot: &domain.Slot{ID: slotID, StartTime: now.Add(-time.Minute)},
		},
		bookings: mockBookingRepo{
			active: &domain.Booking{ID: uuid.New(), SlotID: slotID, UserID: uuid.New(), Status: domain.BookingStatusActive},
		},
		now: func() time.Time { return now },
	}

	_, err := svc.JoinWaitlist(context.Background(), domain.User{ID: userID, Role: domain.RoleUser}, slotID)
	de, ok := domain.AsDomainError(err)
	require.True(t, ok)
	require.Equal(t, domain.ErrorInvalidRequest, de.Code)
}

func TestJoinWaitlist_OwnBookingRejected(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 3, 25, 12, 0, 0, 0, time.UTC)
	slotID := uuid.New()
	userID := uuid.New()
	svc := &Service{
		tx:        mockTx{},
		waitlists: mockWaitlistRepo{},
		slots: mockSlotRepo{
			slot: &domain.Slot{ID: slotID, StartTime: now.Add(time.Hour)},
		},
		bookings: mockBookingRepo{
			active: &domain.Booking{ID: uuid.New(), SlotID: slotID, UserID: userID, Status: domain.BookingStatusActive},
		},
		now: func() time.Time { return now },
	}

	_, err := svc.JoinWaitlist(context.Background(), domain.User{ID: userID, Role: domain.RoleUser}, slotID)
	de, ok := domain.AsDomainError(err)
	require.True(t, ok)
	require.Equal(t, domain.ErrorInvalidRequest, de.Code)
}

func TestLeaveWaitlist_IdempotentAlreadyCancelled(t *testing.T) {
	t.Parallel()

	entryID := uuid.New()
	userID := uuid.New()
	cancelled := domain.WaitlistEntry{
		ID:        entryID,
		SlotID:    uuid.New(),
		UserID:    userID,
		Status:    domain.WaitlistStatusCancelled,
		Position:  10,
		CreatedAt: time.Now().UTC(),
	}
	svc := &Service{
		tx: mockTx{},
		waitlists: mockWaitlistRepo{
			leaveFn: func(ctx context.Context, gotEntryID, gotUserID uuid.UUID) (*domain.WaitlistEntry, bool, error) {
				require.Equal(t, entryID, gotEntryID)
				require.Equal(t, userID, gotUserID)
				return &cancelled, false, nil
			},
		},
		slots:    mockSlotRepo{},
		bookings: mockBookingRepo{},
	}

	entry, err := svc.LeaveWaitlist(context.Background(), domain.User{ID: userID, Role: domain.RoleUser}, entryID)
	require.NoError(t, err)
	require.Equal(t, domain.WaitlistStatusCancelled, entry.Status)
}
