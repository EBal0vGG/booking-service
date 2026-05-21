package waitlist

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
	tx        repository.TxManager
	waitlists repository.WaitlistRepository
	slots     repository.SlotRepository
	bookings  repository.BookingRepository
	now       func() time.Time
}

func NewService(
	tx repository.TxManager,
	waitlists repository.WaitlistRepository,
	slots repository.SlotRepository,
	bookings repository.BookingRepository,
) *Service {
	return &Service{
		tx:        tx,
		waitlists: waitlists,
		slots:     slots,
		bookings:  bookings,
		now:       func() time.Time { return time.Now().UTC() },
	}
}

var _ usecase.WaitlistUsecase = (*Service)(nil)

func (s *Service) JoinWaitlist(ctx context.Context, user domain.User, slotID uuid.UUID) (domain.WaitlistEntry, error) {
	if user.Role != domain.RoleUser {
		return domain.WaitlistEntry{}, domain.NewDomainError(domain.ErrorForbidden, "waitlist is allowed only for user role")
	}

	var joined domain.WaitlistEntry
	err := s.tx.WithinTransaction(ctx, func(txCtx context.Context) error {
		slot, err := s.slots.GetByID(txCtx, slotID)
		if err != nil {
			return domain.WrapDomainError(domain.ErrorInternalError, "get slot", err)
		}
		if slot == nil {
			return domain.NewDomainError(domain.ErrorSlotNotFound, "slot not found")
		}
		if slot.StartTime.UTC().Before(s.now()) {
			return domain.NewDomainError(domain.ErrorInvalidRequest, "cannot join waitlist for past slot")
		}

		activeBooking, err := s.bookings.GetActiveBySlotForUpdate(txCtx, slotID)
		if err != nil {
			return domain.WrapDomainError(domain.ErrorInternalError, "check slot booking", err)
		}
		if activeBooking == nil {
			return domain.NewDomainError(domain.ErrorSlotNotBooked, "slot is not booked")
		}
		if activeBooking.UserID == user.ID {
			return domain.NewDomainError(domain.ErrorInvalidRequest, "cannot join waitlist for own booking")
		}

		entry := domain.WaitlistEntry{
			ID:        uuid.New(),
			SlotID:    slotID,
			UserID:    user.ID,
			Status:    domain.WaitlistStatusActive,
			CreatedAt: s.now(),
		}
		created, err := s.waitlists.Join(txCtx, entry)
		if err != nil {
			if de, ok := domain.AsDomainError(err); ok && de.Code == domain.ErrorWaitlistJoined {
				return err
			}
			return domain.WrapDomainError(domain.ErrorInternalError, "join waitlist", err)
		}
		joined = *created
		return nil
	})
	if err != nil {
		return domain.WaitlistEntry{}, err
	}
	observabilitymetrics.IncWaitlistJoined()
	slog.Info(
		"waitlist_joined",
		"slot_id", joined.SlotID,
		"user_id", joined.UserID,
		"waitlist_entry_id", joined.ID,
		"position", joined.Position,
	)
	return joined, nil
}

func (s *Service) LeaveWaitlist(ctx context.Context, user domain.User, entryID uuid.UUID) (domain.WaitlistEntry, error) {
	if user.Role != domain.RoleUser {
		return domain.WaitlistEntry{}, domain.NewDomainError(domain.ErrorForbidden, "waitlist is allowed only for user role")
	}

	var left domain.WaitlistEntry
	var changed bool
	err := s.tx.WithinTransaction(ctx, func(txCtx context.Context) error {
		entry, didChange, err := s.waitlists.Leave(txCtx, entryID, user.ID)
		if err != nil {
			return domain.WrapDomainError(domain.ErrorInternalError, "leave waitlist", err)
		}
		if entry == nil {
			return domain.NewDomainError(domain.ErrorWaitlistNotFound, "waitlist entry not found")
		}
		changed = didChange
		left = *entry
		return nil
	})
	if err != nil {
		return domain.WaitlistEntry{}, err
	}
	if changed {
		observabilitymetrics.IncWaitlistCancelled()
	}
	return left, nil
}
