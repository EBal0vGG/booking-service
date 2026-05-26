package reservation

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

const (
	defaultReservationTTL       = 5 * time.Minute
	defaultExpirationBatchLimit = 100
)

type Service struct {
	tx             repository.TxManager
	reservations   repository.ReservationRepository
	bookings       repository.BookingRepository
	slots          repository.SlotRepository
	waitlists      repository.WaitlistRepository
	events         EventPublisher
	metrics        ReservationMetrics
	reservationTTL time.Duration
	now            func() time.Time
}

type EventPublisher interface {
	SlotBooked(ctx context.Context, roomID, slotID, bookingID uuid.UUID)
	SlotReleased(ctx context.Context, roomID, slotID, bookingID uuid.UUID)
	SlotAvailable(ctx context.Context, roomID, slotID uuid.UUID)
	SlotReserved(ctx context.Context, roomID, slotID, reservationID uuid.UUID)
	SlotReservationExpired(ctx context.Context, roomID, slotID, reservationID uuid.UUID)
	WaitlistSlotReserved(ctx context.Context, roomID, slotID, userID, reservationID, waitlistEntryID uuid.UUID, expiresAt time.Time)
	ReservationExpired(ctx context.Context, roomID, slotID, userID, reservationID uuid.UUID)
}

type ReservationMetrics interface {
	IncReservationCreated()
	IncReservationConfirmed()
	IncReservationExpired()
	IncReservationCancelled()
}

type noopEventPublisher struct{}
type noopReservationMetrics struct{}

func (noopEventPublisher) SlotBooked(ctx context.Context, roomID, slotID, bookingID uuid.UUID) {
	_, _, _, _ = ctx, roomID, slotID, bookingID
}

func (noopEventPublisher) SlotReleased(ctx context.Context, roomID, slotID, bookingID uuid.UUID) {
	_, _, _, _ = ctx, roomID, slotID, bookingID
}

func (noopEventPublisher) SlotAvailable(ctx context.Context, roomID, slotID uuid.UUID) {
	_, _, _ = ctx, roomID, slotID
}

func (noopEventPublisher) SlotReserved(ctx context.Context, roomID, slotID, reservationID uuid.UUID) {
	_, _, _, _ = ctx, roomID, slotID, reservationID
}

func (noopEventPublisher) SlotReservationExpired(ctx context.Context, roomID, slotID, reservationID uuid.UUID) {
	_, _, _, _ = ctx, roomID, slotID, reservationID
}

func (noopEventPublisher) WaitlistSlotReserved(ctx context.Context, roomID, slotID, userID, reservationID, waitlistEntryID uuid.UUID, expiresAt time.Time) {
	_, _, _, _, _, _, _ = ctx, roomID, slotID, userID, reservationID, waitlistEntryID, expiresAt
}

func (noopEventPublisher) ReservationExpired(ctx context.Context, roomID, slotID, userID, reservationID uuid.UUID) {
	_, _, _, _, _ = ctx, roomID, slotID, userID, reservationID
}

func (noopReservationMetrics) IncReservationCreated()   {}
func (noopReservationMetrics) IncReservationConfirmed() {}
func (noopReservationMetrics) IncReservationExpired()   {}
func (noopReservationMetrics) IncReservationCancelled() {}

func NewService(
	tx repository.TxManager,
	reservations repository.ReservationRepository,
	bookings repository.BookingRepository,
	slots repository.SlotRepository,
	waitlists repository.WaitlistRepository,
	events EventPublisher,
	reservationTTL time.Duration,
) *Service {
	publisher := events
	if publisher == nil {
		publisher = noopEventPublisher{}
	}
	if reservationTTL <= 0 {
		reservationTTL = defaultReservationTTL
	}
	return &Service{
		tx:             tx,
		reservations:   reservations,
		bookings:       bookings,
		slots:          slots,
		waitlists:      waitlists,
		events:         publisher,
		metrics:        noopReservationMetrics{},
		reservationTTL: reservationTTL,
		now:            func() time.Time { return time.Now().UTC() },
	}
}

var _ usecase.ReservationUsecase = (*Service)(nil)
var _ usecase.ReservationExpirationUsecase = (*Service)(nil)

func (s *Service) WithMetrics(metrics ReservationMetrics) *Service {
	if metrics != nil {
		s.metrics = metrics
	}
	return s
}

func (s *Service) reservationMetrics() ReservationMetrics {
	if s.metrics == nil {
		return noopReservationMetrics{}
	}
	return s.metrics
}

func (s *Service) ListMyActiveReservations(ctx context.Context, user domain.User) ([]domain.SlotReservation, error) {
	if user.Role != domain.RoleUser {
		return nil, domain.NewDomainError(domain.ErrorForbidden, "list active reservations is allowed only for user role")
	}

	items, err := s.reservations.ListActiveByUser(ctx, user.ID, s.now())
	if err != nil {
		return nil, domain.WrapDomainError(domain.ErrorInternalError, "list active reservations", err)
	}

	return items, nil
}

func (s *Service) ConfirmReservation(
	ctx context.Context,
	user domain.User,
	reservationID uuid.UUID,
) (domain.Booking, domain.SlotReservation, error) {
	if user.Role != domain.RoleUser {
		return domain.Booking{}, domain.SlotReservation{}, domain.NewDomainError(domain.ErrorForbidden, "confirm reservation is allowed only for user role")
	}

	now := s.now()
	var createdBooking domain.Booking
	var confirmed domain.SlotReservation
	var roomID uuid.UUID
	err := s.tx.WithinTransaction(ctx, func(txCtx context.Context) error {
		existing, err := s.reservations.GetByIDForUpdate(txCtx, reservationID)
		if err != nil {
			return domain.WrapDomainError(domain.ErrorInternalError, "get reservation", err)
		}
		if existing == nil {
			return domain.NewDomainError(domain.ErrorReservationNotFound, "reservation not found")
		}
		if existing.UserID != user.ID {
			return domain.NewDomainError(domain.ErrorForbidden, "cannot confirm another user's reservation")
		}
		if existing.Status != domain.ReservationStatusActive {
			return domain.NewDomainError(domain.ErrorInvalidRequest, "reservation is not active")
		}
		if !existing.ExpiresAt.After(now) {
			return domain.NewDomainError(domain.ErrorInvalidRequest, "reservation is expired")
		}

		activeBooking, err := s.bookings.GetActiveBySlotForUpdate(txCtx, existing.SlotID)
		if err != nil {
			return domain.WrapDomainError(domain.ErrorInternalError, "check active booking", err)
		}
		if activeBooking != nil {
			return domain.NewDomainError(domain.ErrorSlotAlreadyBooked, "slot already booked")
		}

		slot, err := s.slots.GetByID(txCtx, existing.SlotID)
		if err != nil {
			return domain.WrapDomainError(domain.ErrorInternalError, "get slot", err)
		}
		if slot == nil {
			return domain.NewDomainError(domain.ErrorSlotNotFound, "slot not found")
		}
		if slot.StartTime.UTC().Before(now) {
			return domain.NewDomainError(domain.ErrorInvalidRequest, "cannot confirm reservation for past slot")
		}
		roomID = slot.RoomID

		createdBooking = domain.Booking{
			ID:        uuid.New(),
			UserID:    user.ID,
			SlotID:    existing.SlotID,
			Status:    domain.BookingStatusActive,
			CreatedAt: now,
		}
		if err := s.bookings.Create(txCtx, createdBooking); err != nil {
			if de, ok := domain.AsDomainError(err); ok && de.Code == domain.ErrorSlotAlreadyBooked {
				return err
			}
			return domain.WrapDomainError(domain.ErrorInternalError, "create booking", err)
		}

		updated, err := s.reservations.SetConfirmed(txCtx, existing.ID, now)
		if err != nil {
			return domain.WrapDomainError(domain.ErrorInternalError, "confirm reservation", err)
		}
		if updated == nil {
			return domain.NewDomainError(domain.ErrorReservationNotFound, "reservation not found")
		}
		confirmed = *updated
		return nil
	})
	if err != nil {
		return domain.Booking{}, domain.SlotReservation{}, err
	}

	s.reservationMetrics().IncReservationConfirmed()
	if roomID != uuid.Nil {
		s.events.SlotBooked(ctx, roomID, createdBooking.SlotID, createdBooking.ID)
	}
	return createdBooking, confirmed, nil
}

func (s *Service) CancelReservation(
	ctx context.Context,
	user domain.User,
	reservationID uuid.UUID,
) (domain.SlotReservation, error) {
	if user.Role != domain.RoleUser {
		return domain.SlotReservation{}, domain.NewDomainError(domain.ErrorForbidden, "cancel reservation is allowed only for user role")
	}

	now := s.now()
	var cancelled domain.SlotReservation
	var changed bool
	var roomID uuid.UUID
	var notifiedEntry *domain.WaitlistEntry
	var nextReservation *domain.SlotReservation
	err := s.tx.WithinTransaction(ctx, func(txCtx context.Context) error {
		existing, err := s.reservations.GetByIDForUpdate(txCtx, reservationID)
		if err != nil {
			return domain.WrapDomainError(domain.ErrorInternalError, "get reservation", err)
		}
		if existing == nil {
			return domain.NewDomainError(domain.ErrorReservationNotFound, "reservation not found")
		}
		if existing.UserID != user.ID {
			return domain.NewDomainError(domain.ErrorForbidden, "cannot cancel another user's reservation")
		}

		slot, err := s.slots.GetByID(txCtx, existing.SlotID)
		if err != nil {
			return domain.WrapDomainError(domain.ErrorInternalError, "get slot", err)
		}
		if slot != nil {
			roomID = slot.RoomID
		}

		if existing.Status != domain.ReservationStatusActive {
			cancelled = *existing
			return nil
		}

		updated, err := s.reservations.SetCancelled(txCtx, existing.ID)
		if err != nil {
			return domain.WrapDomainError(domain.ErrorInternalError, "cancel reservation", err)
		}
		if updated == nil {
			return domain.NewDomainError(domain.ErrorReservationNotFound, "reservation not found")
		}
		cancelled = *updated
		changed = true

		if s.waitlists == nil {
			return nil
		}

		entry, err := s.waitlists.ClaimNextForNotify(txCtx, existing.SlotID)
		if err != nil {
			return domain.WrapDomainError(domain.ErrorInternalError, "claim next waitlist entry", err)
		}
		notifiedEntry = entry
		if entry == nil {
			return nil
		}

		waitlistEntryID := entry.ID
		nextReservation, err = s.reservations.Create(txCtx, domain.SlotReservation{
			ID:              uuid.New(),
			SlotID:          existing.SlotID,
			UserID:          entry.UserID,
			WaitlistEntryID: &waitlistEntryID,
			Status:          domain.ReservationStatusActive,
			ExpiresAt:       now.Add(s.reservationTTLValue()),
			CreatedAt:       now,
		})
		if err != nil {
			return domain.WrapDomainError(domain.ErrorInternalError, "create next reservation", err)
		}
		return nil
	})
	if err != nil {
		return domain.SlotReservation{}, err
	}
	if !changed {
		return cancelled, nil
	}

	s.reservationMetrics().IncReservationCancelled()
	if nextReservation != nil && notifiedEntry != nil {
		observabilitymetrics.IncWaitlistNotification()
		s.reservationMetrics().IncReservationCreated()
		if roomID != uuid.Nil {
			s.events.SlotReserved(ctx, roomID, cancelled.SlotID, nextReservation.ID)
		}
		s.events.WaitlistSlotReserved(
			ctx,
			roomID,
			cancelled.SlotID,
			notifiedEntry.UserID,
			nextReservation.ID,
			notifiedEntry.ID,
			nextReservation.ExpiresAt,
		)
	} else if roomID != uuid.Nil {
		s.events.SlotAvailable(ctx, roomID, cancelled.SlotID)
	}
	return cancelled, nil
}

func (s *Service) ExpireDue(ctx context.Context, limit int) error {
	if limit <= 0 {
		limit = defaultExpirationBatchLimit
	}

	now := s.now()
	var expired []domain.SlotReservation
	slotRooms := make(map[uuid.UUID]uuid.UUID)
	slotHasReplacement := make(map[uuid.UUID]bool)
	type replacement struct {
		roomID      uuid.UUID
		waitlist    domain.WaitlistEntry
		reservation domain.SlotReservation
	}
	replacements := make([]replacement, 0, limit)

	err := s.tx.WithinTransaction(ctx, func(txCtx context.Context) error {
		updated, err := s.reservations.ExpireBatch(txCtx, now, limit)
		if err != nil {
			return domain.WrapDomainError(domain.ErrorInternalError, "expire reservations", err)
		}
		expired = updated
		for _, item := range expired {
			slot, err := s.slots.GetByID(txCtx, item.SlotID)
			if err != nil {
				return domain.WrapDomainError(domain.ErrorInternalError, "get slot", err)
			}
			if slot != nil {
				slotRooms[item.SlotID] = slot.RoomID
			}

			if s.waitlists == nil {
				continue
			}

			entry, err := s.waitlists.ClaimNextForNotify(txCtx, item.SlotID)
			if err != nil {
				return domain.WrapDomainError(domain.ErrorInternalError, "claim next waitlist entry", err)
			}
			if entry == nil {
				continue
			}
			waitlistEntryID := entry.ID
			created, err := s.reservations.Create(txCtx, domain.SlotReservation{
				ID:              uuid.New(),
				SlotID:          item.SlotID,
				UserID:          entry.UserID,
				WaitlistEntryID: &waitlistEntryID,
				Status:          domain.ReservationStatusActive,
				ExpiresAt:       now.Add(s.reservationTTLValue()),
				CreatedAt:       now,
			})
			if err != nil {
				return domain.WrapDomainError(domain.ErrorInternalError, "create reservation for next waitlist user", err)
			}

			replacements = append(replacements, replacement{
				roomID:      slotRooms[item.SlotID],
				waitlist:    *entry,
				reservation: *created,
			})
			slotHasReplacement[item.SlotID] = true
		}
		return nil
	})
	if err != nil {
		return err
	}
	if len(expired) == 0 {
		return nil
	}

	for _, item := range expired {
		s.reservationMetrics().IncReservationExpired()
		roomID := slotRooms[item.SlotID]
		s.events.ReservationExpired(ctx, roomID, item.SlotID, item.UserID, item.ID)
		if roomID != uuid.Nil {
			s.events.SlotReservationExpired(ctx, roomID, item.SlotID, item.ID)
			if !slotHasReplacement[item.SlotID] {
				s.events.SlotAvailable(ctx, roomID, item.SlotID)
			}
		}
	}

	for _, item := range replacements {
		observabilitymetrics.IncWaitlistNotification()
		s.reservationMetrics().IncReservationCreated()
		if item.roomID != uuid.Nil {
			s.events.SlotReserved(ctx, item.roomID, item.reservation.SlotID, item.reservation.ID)
		}
		s.events.WaitlistSlotReserved(
			ctx,
			item.roomID,
			item.reservation.SlotID,
			item.waitlist.UserID,
			item.reservation.ID,
			item.waitlist.ID,
			item.reservation.ExpiresAt,
		)
		slog.Info(
			"waitlist_reserved_after_expire",
			"slot_id", item.reservation.SlotID,
			"user_id", item.waitlist.UserID,
			"waitlist_entry_id", item.waitlist.ID,
			"reservation_id", item.reservation.ID,
			"expires_at", item.reservation.ExpiresAt,
		)
	}

	return nil
}

func (s *Service) reservationTTLValue() time.Duration {
	if s.reservationTTL > 0 {
		return s.reservationTTL
	}
	return defaultReservationTTL
}
