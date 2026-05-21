//go:build integration

package integrationtest

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"booking-service/internal/domain"
	"booking-service/internal/repository/postgres"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
)

func TestIntegration_WaitlistDeterministicOrderByPosition(t *testing.T) {
	ctx := context.Background()
	pool := setupPool(t)
	waitlistRepo := postgres.NewWaitlistRepo(pool)

	slotID := createWaitlistTestSlot(ctx, t, pool)
	userIDs := []uuid.UUID{uuid.New(), uuid.New(), uuid.New()}
	insertWaitlistUsers(ctx, t, pool, userIDs)

	now := time.Now().UTC()
	first, err := waitlistRepo.Join(ctx, domain.WaitlistEntry{
		ID:        uuid.New(),
		SlotID:    slotID,
		UserID:    userIDs[0],
		Status:    domain.WaitlistStatusActive,
		CreatedAt: now,
	})
	require.NoError(t, err)
	second, err := waitlistRepo.Join(ctx, domain.WaitlistEntry{
		ID:        uuid.New(),
		SlotID:    slotID,
		UserID:    userIDs[1],
		Status:    domain.WaitlistStatusActive,
		CreatedAt: now,
	})
	require.NoError(t, err)
	third, err := waitlistRepo.Join(ctx, domain.WaitlistEntry{
		ID:        uuid.New(),
		SlotID:    slotID,
		UserID:    userIDs[2],
		Status:    domain.WaitlistStatusActive,
		CreatedAt: now,
	})
	require.NoError(t, err)

	require.Less(t, first.Position, second.Position)
	require.Less(t, second.Position, third.Position)

	claim1, err := waitlistRepo.ClaimNextForNotify(ctx, slotID)
	require.NoError(t, err)
	require.NotNil(t, claim1)
	require.Equal(t, first.ID, claim1.ID)
	require.Equal(t, domain.WaitlistStatusNotified, claim1.Status)

	claim2, err := waitlistRepo.ClaimNextForNotify(ctx, slotID)
	require.NoError(t, err)
	require.NotNil(t, claim2)
	require.Equal(t, second.ID, claim2.ID)

	claim3, err := waitlistRepo.ClaimNextForNotify(ctx, slotID)
	require.NoError(t, err)
	require.NotNil(t, claim3)
	require.Equal(t, third.ID, claim3.ID)
}

func TestIntegration_WaitlistConcurrentClaimDoesNotDuplicate(t *testing.T) {
	ctx := context.Background()
	pool := setupPool(t)
	waitlistRepo := postgres.NewWaitlistRepo(pool)
	txm := postgres.NewTxManager(pool)

	slotID := createWaitlistTestSlot(ctx, t, pool)
	userID := uuid.New()
	insertWaitlistUsers(ctx, t, pool, []uuid.UUID{userID})
	entry, err := waitlistRepo.Join(ctx, domain.WaitlistEntry{
		ID:        uuid.New(),
		SlotID:    slotID,
		UserID:    userID,
		Status:    domain.WaitlistStatusActive,
		CreatedAt: time.Now().UTC(),
	})
	require.NoError(t, err)

	start := make(chan struct{})
	results := make(chan uuid.UUID, 2)
	var wg sync.WaitGroup
	worker := func() {
		defer wg.Done()
		<-start
		err := txm.WithinTransaction(ctx, func(txCtx context.Context) error {
			claimed, err := waitlistRepo.ClaimNextForNotify(txCtx, slotID)
			if err != nil {
				return err
			}
			if claimed != nil {
				results <- claimed.ID
			}
			return nil
		})
		require.NoError(t, err)
	}

	wg.Add(2)
	go worker()
	go worker()
	close(start)
	wg.Wait()
	close(results)

	var claimedIDs []uuid.UUID
	for id := range results {
		claimedIDs = append(claimedIDs, id)
	}
	require.Len(t, claimedIDs, 1)
	require.Equal(t, entry.ID, claimedIDs[0])
}

func createWaitlistTestSlot(ctx context.Context, t *testing.T, pool *pgxpool.Pool) uuid.UUID {
	t.Helper()

	roomID := uuid.New()
	slotID := uuid.New()
	start := time.Now().UTC().Add(2 * time.Hour).Truncate(time.Minute)
	end := start.Add(30 * time.Minute)

	_, err := pool.Exec(ctx, `
INSERT INTO rooms (id, name, description, capacity, created_at)
VALUES ($1, $2, NULL, 6, NOW())
`, roomID, fmt.Sprintf("waitlist-room-%s", roomID.String()[:8]))
	require.NoError(t, err)

	_, err = pool.Exec(ctx, `
INSERT INTO slots (id, room_id, start_time, end_time, created_at)
VALUES ($1, $2, $3, $4, NOW())
`, slotID, roomID, start, end)
	require.NoError(t, err)

	return slotID
}

func insertWaitlistUsers(ctx context.Context, t *testing.T, pool *pgxpool.Pool, userIDs []uuid.UUID) {
	t.Helper()
	for idx, userID := range userIDs {
		_, err := pool.Exec(ctx, `
INSERT INTO users (id, email, role, created_at, password_hash)
VALUES ($1, $2, 'user', NOW(), NULL)
`, userID, fmt.Sprintf("waitlist-%d@integration.test", idx))
		require.NoError(t, err)
	}
}
