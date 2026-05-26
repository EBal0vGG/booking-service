//go:build integration

package integrationtest

import (
	"context"
	"testing"
	"time"

	"booking-service/internal/domain"
	"booking-service/internal/repository/postgres"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestIntegration_ReservationUniqueActivePerSlot(t *testing.T) {
	ctx := context.Background()
	pool := setupPool(t)
	reservationRepo := postgres.NewReservationRepo(pool)

	slotID := createWaitlistTestSlot(ctx, t, pool)
	userIDs := []uuid.UUID{uuid.New(), uuid.New()}
	insertWaitlistUsers(ctx, t, pool, userIDs)

	now := time.Now().UTC()
	_, err := reservationRepo.Create(ctx, domain.SlotReservation{
		ID:        uuid.New(),
		SlotID:    slotID,
		UserID:    userIDs[0],
		Status:    domain.ReservationStatusActive,
		ExpiresAt: now.Add(5 * time.Minute),
		CreatedAt: now,
	})
	require.NoError(t, err)

	_, err = reservationRepo.Create(ctx, domain.SlotReservation{
		ID:        uuid.New(),
		SlotID:    slotID,
		UserID:    userIDs[1],
		Status:    domain.ReservationStatusActive,
		ExpiresAt: now.Add(10 * time.Minute),
		CreatedAt: now,
	})
	require.Error(t, err)
	de, ok := domain.AsDomainError(err)
	require.True(t, ok)
	require.Equal(t, domain.ErrorSlotReserved, de.Code)
}

func TestIntegration_ReservationExpireBatchExpiresDueOnly(t *testing.T) {
	ctx := context.Background()
	pool := setupPool(t)
	reservationRepo := postgres.NewReservationRepo(pool)

	slotDueID := createWaitlistTestSlot(ctx, t, pool)
	slotFutureID := createWaitlistTestSlot(ctx, t, pool)
	userIDs := []uuid.UUID{uuid.New(), uuid.New()}
	insertWaitlistUsers(ctx, t, pool, userIDs)

	now := time.Now().UTC().Truncate(time.Second)
	dueReservation, err := reservationRepo.Create(ctx, domain.SlotReservation{
		ID:        uuid.New(),
		SlotID:    slotDueID,
		UserID:    userIDs[0],
		Status:    domain.ReservationStatusActive,
		ExpiresAt: now.Add(-time.Minute),
		CreatedAt: now,
	})
	require.NoError(t, err)

	futureReservation, err := reservationRepo.Create(ctx, domain.SlotReservation{
		ID:        uuid.New(),
		SlotID:    slotFutureID,
		UserID:    userIDs[1],
		Status:    domain.ReservationStatusActive,
		ExpiresAt: now.Add(10 * time.Minute),
		CreatedAt: now,
	})
	require.NoError(t, err)

	expired, err := reservationRepo.ExpireBatch(ctx, now, 10)
	require.NoError(t, err)
	require.Len(t, expired, 1)
	require.Equal(t, dueReservation.ID, expired[0].ID)
	require.Equal(t, domain.ReservationStatusExpired, expired[0].Status)
	require.NotNil(t, expired[0].ExpiredAt)

	currentDue, err := reservationRepo.GetByID(ctx, dueReservation.ID)
	require.NoError(t, err)
	require.NotNil(t, currentDue)
	require.Equal(t, domain.ReservationStatusExpired, currentDue.Status)
	require.NotNil(t, currentDue.ExpiredAt)

	currentFuture, err := reservationRepo.GetByID(ctx, futureReservation.ID)
	require.NoError(t, err)
	require.NotNil(t, currentFuture)
	require.Equal(t, domain.ReservationStatusActive, currentFuture.Status)
	require.Nil(t, currentFuture.ExpiredAt)

	expiredAgain, err := reservationRepo.ExpireBatch(ctx, now, 10)
	require.NoError(t, err)
	require.Empty(t, expiredAgain)
}

func TestIntegration_ReservationConfirmVsExpireRaceSingleWinner(t *testing.T) {
	ctx := context.Background()
	pool := setupPool(t)
	reservationRepo := postgres.NewReservationRepo(pool)
	txm := postgres.NewTxManager(pool)

	slotID := createWaitlistTestSlot(ctx, t, pool)
	userID := uuid.New()
	insertWaitlistUsers(ctx, t, pool, []uuid.UUID{userID})

	now := time.Now().UTC().Truncate(time.Second)
	reservation, err := reservationRepo.Create(ctx, domain.SlotReservation{
		ID:        uuid.New(),
		SlotID:    slotID,
		UserID:    userID,
		Status:    domain.ReservationStatusActive,
		ExpiresAt: now.Add(-time.Minute),
		CreatedAt: now,
	})
	require.NoError(t, err)

	locked := make(chan struct{})
	releaseConfirm := make(chan struct{})
	confirmErrCh := make(chan error, 1)

	go func() {
		confirmErrCh <- txm.WithinTransaction(ctx, func(txCtx context.Context) error {
			lockedReservation, err := reservationRepo.GetByIDForUpdate(txCtx, reservation.ID)
			if err != nil {
				return err
			}
			if lockedReservation == nil {
				return domain.NewDomainError(domain.ErrorReservationNotFound, "reservation not found")
			}
			close(locked)
			<-releaseConfirm
			_, err = reservationRepo.SetConfirmed(txCtx, reservation.ID, now.Add(30*time.Second))
			return err
		})
	}()

	<-locked
	err = txm.WithinTransaction(ctx, func(txCtx context.Context) error {
		expired, expireErr := reservationRepo.ExpireBatch(txCtx, now, 10)
		if expireErr != nil {
			return expireErr
		}
		require.Empty(t, expired)
		return nil
	})
	require.NoError(t, err)

	close(releaseConfirm)
	require.NoError(t, <-confirmErrCh)

	current, err := reservationRepo.GetByID(ctx, reservation.ID)
	require.NoError(t, err)
	require.NotNil(t, current)
	require.Equal(t, domain.ReservationStatusConfirmed, current.Status)
	require.NotNil(t, current.ConfirmedAt)
	require.Nil(t, current.ExpiredAt)

	expiredAfterConfirm, err := reservationRepo.ExpireBatch(ctx, now.Add(time.Hour), 10)
	require.NoError(t, err)
	require.Empty(t, expiredAfterConfirm)
}
