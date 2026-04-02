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
	now      func() time.Time
}

func NewService(
	tx repository.TxManager,
	bookings repository.BookingRepository,
	slots repository.SlotRepository,
) *Service {
	return &Service{
		tx:       tx,
		bookings: bookings,
		slots:    slots,
		now:      func() time.Time { return time.Now().UTC() },
	}
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
		return domain.Booking{}, domain.NewDomainError(domain.ErrorForbidden, "booking is allowed only for user role")
	}

	now := s.now()
	var created domain.Booking
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
		return domain.Booking{}, err
	}
	return created, nil
}

func (s *Service) CancelBooking(ctx context.Context, user domain.User, bookingID uuid.UUID) (domain.Booking, error) {
	if user.Role != domain.RoleUser {
		return domain.Booking{}, domain.NewDomainError(domain.ErrorForbidden, "cancel is allowed only for user role")
	}

	var result domain.Booking
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
		result = *updated
		return nil
	})
	if err != nil {
		return domain.Booking{}, err
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

