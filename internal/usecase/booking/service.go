package booking

import (
	"context"
	"log/slog"
	"time"

	"booking-service/internal/domain"
	observabilitymetrics "booking-service/internal/observability/metrics"
	"booking-service/internal/repository"
	usecase "booking-service/internal/usecase"

	"github.com/google/uuid"
)

type Service struct {
	tx             repository.TxManager
	bookings       repository.BookingRepository
	slots          repository.SlotRepository
	waitlists      repository.WaitlistRepository
	reservations   repository.ReservationRepository
	events         EventPublisher
	metrics        BookingMetrics
	reservationTTL time.Duration
	now            func() time.Time
}

type EventPublisher interface {
	SlotBooked(ctx context.Context, roomID, slotID, bookingID uuid.UUID)
	SlotReleased(ctx context.Context, roomID, slotID, bookingID uuid.UUID)
	SlotReserved(ctx context.Context, roomID, slotID, reservationID uuid.UUID)
	WaitlistSlotReserved(ctx context.Context, roomID, slotID, userID, reservationID, waitlistEntryID uuid.UUID, expiresAt time.Time)
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

const defaultReservationTTL = 5 * time.Minute

func (noopEventPublisher) SlotBooked(ctx context.Context, roomID, slotID, bookingID uuid.UUID) {
	_, _, _, _ = ctx, roomID, slotID, bookingID
}

func (noopEventPublisher) SlotReleased(ctx context.Context, roomID, slotID, bookingID uuid.UUID) {
	_, _, _, _ = ctx, roomID, slotID, bookingID
}

func (noopEventPublisher) SlotReserved(ctx context.Context, roomID, slotID, reservationID uuid.UUID) {
	_, _, _, _ = ctx, roomID, slotID, reservationID
}

func (noopEventPublisher) WaitlistSlotReserved(ctx context.Context, roomID, slotID, userID, reservationID, waitlistEntryID uuid.UUID, expiresAt time.Time) {
	_, _, _, _, _, _, _ = ctx, roomID, slotID, userID, reservationID, waitlistEntryID, expiresAt
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
		tx:             tx,
		bookings:       bookings,
		slots:          slots,
		events:         publisher,
		metrics:        noopBookingMetrics{},
		reservationTTL: defaultReservationTTL,
		now:            func() time.Time { return time.Now().UTC() },
	}
}

func (s *Service) WithWaitlistRepository(waitlists repository.WaitlistRepository) *Service {
	s.waitlists = waitlists
	return s
}

func (s *Service) WithReservationRepository(reservations repository.ReservationRepository, ttl time.Duration) *Service {
	s.reservations = reservations
	if ttl > 0 {
		s.reservationTTL = ttl
	}
	return s
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
		if s.reservations != nil {
			activeReservation, err := s.reservations.GetActiveBySlotForUpdate(txCtx, slotID)
			if err != nil {
				return domain.WrapDomainError(domain.ErrorInternalError, "check slot reservation", err)
			}
			if activeReservation != nil {
				return domain.NewDomainError(domain.ErrorSlotReserved, "slot is reserved")
			}
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
		if de, ok := domain.AsDomainError(err); ok && (de.Code == domain.ErrorSlotAlreadyBooked || de.Code == domain.ErrorSlotReserved) {
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
	var slotReleased bool
	var notifiedEntry *domain.WaitlistEntry
	var createdReservation *domain.SlotReservation
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
		wasActive := existing.Status == domain.BookingStatusActive

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
		if wasActive {
			slotReleased = true
			if s.waitlists != nil {
				entry, err := s.waitlists.ClaimNextForNotify(txCtx, updated.SlotID)
				if err != nil {
					return domain.WrapDomainError(domain.ErrorInternalError, "claim waitlist entry", err)
				}
				notifiedEntry = entry
				if entry != nil {
					if s.reservations == nil {
						return domain.NewDomainError(domain.ErrorInternalError, "reservation repository is not configured")
					}
					reservationNow := s.now()
					waitlistEntryID := entry.ID
					createdReservation, err = s.reservations.Create(txCtx, domain.SlotReservation{
						ID:              uuid.New(),
						SlotID:          updated.SlotID,
						UserID:          entry.UserID,
						WaitlistEntryID: &waitlistEntryID,
						Status:          domain.ReservationStatusActive,
						ExpiresAt:       reservationNow.Add(s.reservationTTLValue()),
						CreatedAt:       reservationNow,
					})
					if err != nil {
						return domain.WrapDomainError(domain.ErrorInternalError, "create slot reservation", err)
					}
				}
			}
		}
		result = *updated
		return nil
	})
	if err != nil {
		s.bookingMetrics().IncBookingCancelError()
		return domain.Booking{}, err
	}
	s.bookingMetrics().IncBookingCancelled()
	if slotReleased && roomID != uuid.Nil {
		if createdReservation != nil && notifiedEntry != nil {
			observabilitymetrics.IncWaitlistNotification()
			observabilitymetrics.IncReservationCreated()
			slog.Info(
				"waitlist_reserved",
				"slot_id", notifiedEntry.SlotID,
				"user_id", notifiedEntry.UserID,
				"waitlist_entry_id", notifiedEntry.ID,
				"position", notifiedEntry.Position,
				"reservation_id", createdReservation.ID,
				"expires_at", createdReservation.ExpiresAt,
			)
			s.eventPublisher().SlotReserved(ctx, roomID, result.SlotID, createdReservation.ID)
			s.eventPublisher().WaitlistSlotReserved(
				ctx,
				roomID,
				result.SlotID,
				notifiedEntry.UserID,
				createdReservation.ID,
				notifiedEntry.ID,
				createdReservation.ExpiresAt,
			)
		} else {
			s.eventPublisher().SlotReleased(ctx, roomID, result.SlotID, result.ID)
			slog.Info(
				"waitlist_skipped",
				"slot_id", result.SlotID,
				"user_id", uuid.Nil,
				"waitlist_entry_id", uuid.Nil,
				"position", 0,
				"reason", "empty_queue",
			)
		}
	}
	return result, nil
}

func (s *Service) reservationTTLValue() time.Duration {
	if s.reservationTTL > 0 {
		return s.reservationTTL
	}
	return defaultReservationTTL
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
