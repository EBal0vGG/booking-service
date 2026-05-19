package booking

import (
	"context"
	"time"

	"booking-service/internal/domain"
	"booking-service/internal/repository"
	usecase "booking-service/internal/usecase"

	"github.com/google/uuid"
)

type Service struct {
	tx       repository.TxManager
	bookings repository.BookingRepository
	slots    repository.SlotRepository
	events   EventPublisher
	metrics  BookingMetrics
	now      func() time.Time
}

type EventPublisher interface {
	SlotBooked(ctx context.Context, roomID, slotID, bookingID uuid.UUID)
	SlotReleased(ctx context.Context, roomID, slotID, bookingID uuid.UUID)
}

type BookingMetrics interface {
	IncBookingCreated()
	IncBookingCancelled()
	IncBookingConflict()
	IncBookingCreateError()
	IncBookingCancelError()
}

type noopEventPublisher struct{}
type noopBookingMetrics struct{}

func (noopEventPublisher) SlotBooked(ctx context.Context, roomID, slotID, bookingID uuid.UUID) {
	_, _, _, _ = ctx, roomID, slotID, bookingID
}

func (noopEventPublisher) SlotReleased(ctx context.Context, roomID, slotID, bookingID uuid.UUID) {
	_, _, _, _ = ctx, roomID, slotID, bookingID
}

func (noopBookingMetrics) IncBookingCreated()     {}
func (noopBookingMetrics) IncBookingCancelled()   {}
func (noopBookingMetrics) IncBookingConflict()    {}
func (noopBookingMetrics) IncBookingCreateError() {}
func (noopBookingMetrics) IncBookingCancelError() {}

func (s *Service) eventPublisher() EventPublisher {
	if s.events == nil {
		return noopEventPublisher{}
	}
	return s.events
}

func (s *Service) bookingMetrics() BookingMetrics {
	if s.metrics == nil {
		return noopBookingMetrics{}
	}
	return s.metrics
}

func NewService(
	tx repository.TxManager,
	bookings repository.BookingRepository,
	slots repository.SlotRepository,
	events ...EventPublisher,
) *Service {
	publisher := EventPublisher(noopEventPublisher{})
	if len(events) > 0 && events[0] != nil {
		publisher = events[0]
	}
	return &Service{
		tx:       tx,
		bookings: bookings,
		slots:    slots,
		events:   publisher,
		metrics:  noopBookingMetrics{},
		now:      func() time.Time { return time.Now().UTC() },
	}
}

func (s *Service) WithMetrics(metrics BookingMetrics) *Service {
	if metrics != nil {
		s.metrics = metrics
	}
	return s
}

var _ usecase.BookingUsecase = (*Service)(nil)

func (s *Service) CreateBooking(
	ctx context.Context,
	user domain.User,
	slotID uuid.UUID,
	createConferenceLink bool,
) (domain.Booking, error) {
	_ = createConferenceLink // conference integration is intentionally postponed

	if user.Role != domain.RoleUser {
		s.bookingMetrics().IncBookingCreateError()
		return domain.Booking{}, domain.NewDomainError(domain.ErrorForbidden, "booking is allowed only for user role")
	}

	now := s.now()
	var created domain.Booking
	var roomID uuid.UUID
	err := s.tx.WithinTransaction(ctx, func(txCtx context.Context) error {
		slot, err := s.slots.GetByID(txCtx, slotID)
		if err != nil {
			return domain.WrapDomainError(domain.ErrorInternalError, "get slot", err)
		}
		if slot == nil {
			return domain.NewDomainError(domain.ErrorSlotNotFound, "slot not found")
		}
		if slot.StartTime.UTC().Before(now) {
			return domain.NewDomainError(domain.ErrorInvalidRequest, "cannot create booking for past slot")
		}
		roomID = slot.RoomID

		created = domain.Booking{
			ID:        uuid.New(),
			UserID:    user.ID,
			SlotID:    slotID,
			Status:    domain.BookingStatusActive,
			CreatedAt: now,
		}
		if createConferenceLink {
			// External conference link integration is intentionally omitted at this stage.
			created.ConferenceLink = nil
		}

		if err := s.bookings.Create(txCtx, created); err != nil {
			// Repo must map DB unique violation (active slot conflict) into DomainError SLOT_ALREADY_BOOKED.
			if de, ok := domain.AsDomainError(err); ok && de.Code == domain.ErrorSlotAlreadyBooked {
				return err
			}
			return domain.WrapDomainError(domain.ErrorInternalError, "create booking", err)
		}
		return nil
	})
	if err != nil {
		if de, ok := domain.AsDomainError(err); ok && de.Code == domain.ErrorSlotAlreadyBooked {
			s.bookingMetrics().IncBookingConflict()
		} else {
			s.bookingMetrics().IncBookingCreateError()
		}
		return domain.Booking{}, err
	}
	s.bookingMetrics().IncBookingCreated()
	s.eventPublisher().SlotBooked(ctx, roomID, created.SlotID, created.ID)
	return created, nil
}

func (s *Service) CancelBooking(ctx context.Context, user domain.User, bookingID uuid.UUID) (domain.Booking, error) {
	if user.Role != domain.RoleUser {
		s.bookingMetrics().IncBookingCancelError()
		return domain.Booking{}, domain.NewDomainError(domain.ErrorForbidden, "cancel is allowed only for user role")
	}

	var result domain.Booking
	var roomID uuid.UUID
	err := s.tx.WithinTransaction(ctx, func(txCtx context.Context) error {
		existing, err := s.bookings.GetByID(txCtx, bookingID)
		if err != nil {
			return domain.WrapDomainError(domain.ErrorInternalError, "get booking", err)
		}
		if existing == nil {
			return domain.NewDomainError(domain.ErrorBookingNotFound, "booking not found")
		}
		if existing.UserID != user.ID {
			return domain.NewDomainError(domain.ErrorForbidden, "cannot cancel another user's booking")
		}

		updated, err := s.bookings.SetCancelled(txCtx, bookingID)
		if err != nil {
			return domain.WrapDomainError(domain.ErrorInternalError, "cancel booking", err)
		}
		if updated == nil {
			return domain.NewDomainError(domain.ErrorBookingNotFound, "booking not found")
		}
		slot, err := s.slots.GetByID(txCtx, updated.SlotID)
		if err != nil {
			return domain.WrapDomainError(domain.ErrorInternalError, "get slot", err)
		}
		if slot != nil {
			roomID = slot.RoomID
		}
		result = *updated
		return nil
	})
	if err != nil {
		s.bookingMetrics().IncBookingCancelError()
		return domain.Booking{}, err
	}
	s.bookingMetrics().IncBookingCancelled()
	if roomID != uuid.Nil {
		s.eventPublisher().SlotReleased(ctx, roomID, result.SlotID, result.ID)
	}
	return result, nil
}

func (s *Service) ListBookings(ctx context.Context, user domain.User, page, pageSize int) ([]domain.Booking, domain.Pagination, error) {
	if user.Role != domain.RoleAdmin {
		return nil, domain.Pagination{}, domain.NewDomainError(domain.ErrorForbidden, "admin only")
	}
	if page <= 0 || pageSize <= 0 {
		return nil, domain.Pagination{}, domain.NewDomainError(domain.ErrorInvalidRequest, "page and pageSize must be > 0")
	}
	if pageSize > 100 {
		return nil, domain.Pagination{}, domain.NewDomainError(domain.ErrorInvalidRequest, "pageSize must be <= 100")
	}

	bookings, total, err := s.bookings.List(ctx, page, pageSize)
	if err != nil {
		return nil, domain.Pagination{}, domain.WrapDomainError(domain.ErrorInternalError, "list bookings", err)
	}
	return bookings, domain.Pagination{Page: page, PageSize: pageSize, Total: total}, nil
}

func (s *Service) ListMyBookings(ctx context.Context, user domain.User) ([]domain.Booking, error) {
	if user.Role != domain.RoleUser {
		return nil, domain.NewDomainError(domain.ErrorForbidden, "only user role can access own bookings")
	}
	bookings, err := s.bookings.ListFutureByUser(ctx, user.ID, s.now())
	if err != nil {
		return nil, domain.WrapDomainError(domain.ErrorInternalError, "list my bookings", err)
	}
	return bookings, nil
}
