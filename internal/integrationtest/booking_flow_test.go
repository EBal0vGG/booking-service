//go:build integration

package integrationtest

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"booking-service/internal/app"
	"booking-service/internal/app/slotgen"
	"booking-service/internal/domain"
	"booking-service/internal/repository/postgres"
	usecase "booking-service/internal/usecase"
	bookinguc "booking-service/internal/usecase/booking"
	roomuc "booking-service/internal/usecase/room"
	scheduleuc "booking-service/internal/usecase/schedule"
	slotuc "booking-service/internal/usecase/slot"
	waitlistuc "booking-service/internal/usecase/waitlist"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
)

// DummyLogin user IDs (must exist in users for FK).
var (
	userID  = uuid.MustParse("22222222-2222-2222-2222-222222222222")
	adminID = uuid.MustParse("11111111-1111-1111-1111-111111111111")
)

func dsn(t *testing.T) string {
	t.Helper()
	d := os.Getenv("TEST_DATABASE_URL")
	if d == "" {
		d = os.Getenv("DATABASE_URL")
	}
	if d == "" {
		t.Skip("set TEST_DATABASE_URL or DATABASE_URL (e.g. postgres://booking:booking@localhost:5432/booking?sslmode=disable)")
	}
	return d
}

func skipIfNoPostgres(t *testing.T, err error) {
	t.Helper()
	if err == nil {
		return
	}
	t.Skipf("postgres not reachable (start docker compose or export TEST_DATABASE_URL): %v", err)
}

// migrationsPath returns an absolute file:// URL to the repo migrations/ directory,
// independent of the process working directory when running `go test`.
func migrationsPath(t *testing.T) string {
	t.Helper()
	_, filename, _, ok := runtime.Caller(0)
	require.True(t, ok)
	dir := filepath.Dir(filename)
	p := filepath.Join(dir, "..", "..", "migrations")
	abs, err := filepath.Abs(p)
	require.NoError(t, err)
	return "file://" + filepath.ToSlash(abs)
}

func isoWeekdayUTC(t time.Time) int {
	w := t.UTC().Weekday()
	if w == time.Sunday {
		return 7
	}
	return int(w)
}

func cleanTables(ctx context.Context, t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	_, err := pool.Exec(ctx, `
DELETE FROM waitlist_entries;
DELETE FROM bookings;
DELETE FROM slots;
DELETE FROM schedules;
DELETE FROM rooms;
DELETE FROM users;
`)
	require.NoError(t, err)
}

func seedUsers(ctx context.Context, t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	_, err := pool.Exec(ctx, `
INSERT INTO users (id, email, role, created_at) VALUES
  ($1, 'admin@integration.test', 'admin', NOW()),
  ($2, 'user@integration.test', 'user', NOW())
ON CONFLICT (id) DO UPDATE SET email = EXCLUDED.email;
`, adminID, userID)
	require.NoError(t, err)
}

func setupPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	ctx := context.Background()
	url := dsn(t)
	pool, err := postgres.NewPool(ctx, url)
	skipIfNoPostgres(t, err)
	t.Cleanup(func() { pool.Close() })

	require.NoError(t, app.RunMigrations(migrationsPath(t), url))
	cleanTables(ctx, t, pool)
	seedUsers(ctx, t, pool)
	return pool
}

func TestIntegration_FullBookingFlow(t *testing.T) {
	ctx := context.Background()
	pool := setupPool(t)

	roomRepo := postgres.NewRoomRepo(pool)
	scheduleRepo := postgres.NewScheduleRepo(pool)
	slotRepo := postgres.NewSlotRepo(pool)
	bookingRepo := postgres.NewBookingRepo(pool)
	txm := postgres.NewTxManager(pool)

	roomSvc := roomuc.NewService(roomRepo)
	scheduleSvc := scheduleuc.NewService(roomRepo, scheduleRepo)
	slotSvc := slotuc.NewService(roomRepo, slotRepo)
	bookingSvc := bookinguc.NewService(txm, bookingRepo, slotRepo)
	gen := slotgen.NewGenerator(roomRepo, scheduleRepo, slotRepo)

	admin := domain.User{ID: adminID, Role: domain.RoleAdmin, Email: ""}
	regular := domain.User{ID: userID, Role: domain.RoleUser, Email: ""}

	room, err := roomSvc.CreateRoom(ctx, admin, usecase.RoomCreateInput{Name: "integration-room"})
	require.NoError(t, err)

	// Pick a calendar day in the rolling window (14d) so generated slots are in the future.
	nextDay := time.Now().UTC().Add(36 * time.Hour)
	listDate := time.Date(nextDay.Year(), nextDay.Month(), nextDay.Day(), 0, 0, 0, 0, time.UTC)
	dow := isoWeekdayUTC(listDate)

	_, err = scheduleSvc.CreateSchedule(ctx, admin, room.ID, usecase.CreateScheduleInput{
		DaysOfWeek: []int{dow},
		StartTime:  "09:00",
		EndTime:    "12:00",
	})
	require.NoError(t, err)

	require.NoError(t, gen.Generate(ctx))

	before, err := slotSvc.ListAvailableSlots(ctx, regular, room.ID, listDate)
	require.NoError(t, err)
	require.NotEmpty(t, before, "no free slots — check DB, migrations, and date/weekday alignment")
	require.True(t, before[0].StartTime.After(time.Now().UTC()), "slot start must be in the future so CreateBooking does not flake on time")
	targetSlotID := before[0].ID

	b1, err := bookingSvc.CreateBooking(ctx, regular, targetSlotID, false)
	require.NoError(t, err)

	afterBook, err := slotSvc.ListAvailableSlots(ctx, regular, room.ID, listDate)
	require.NoError(t, err)
	require.False(t, containsSlotID(afterBook, targetSlotID), "booked slot must disappear from available list")

	_, err = bookingSvc.CancelBooking(ctx, regular, b1.ID)
	require.NoError(t, err)

	afterCancel, err := slotSvc.ListAvailableSlots(ctx, regular, room.ID, listDate)
	require.NoError(t, err)
	require.True(t, containsSlotID(afterCancel, targetSlotID), "after cancel the slot must be available again")

	b2, err := bookingSvc.CreateBooking(ctx, regular, targetSlotID, false)
	require.NoError(t, err)

	cancelledOnce, err := bookingSvc.CancelBooking(ctx, regular, b2.ID)
	require.NoError(t, err)
	require.Equal(t, domain.BookingStatusCancelled, cancelledOnce.Status)

	cancelledTwice, err := bookingSvc.CancelBooking(ctx, regular, b2.ID)
	require.NoError(t, err)
	require.Equal(t, domain.BookingStatusCancelled, cancelledTwice.Status)
	require.Equal(t, b2.ID, cancelledTwice.ID)
}

func TestIntegration_DoubleBookSameSlot(t *testing.T) {
	ctx := context.Background()
	pool := setupPool(t)

	roomRepo := postgres.NewRoomRepo(pool)
	scheduleRepo := postgres.NewScheduleRepo(pool)
	slotRepo := postgres.NewSlotRepo(pool)
	bookingRepo := postgres.NewBookingRepo(pool)
	txm := postgres.NewTxManager(pool)

	roomSvc := roomuc.NewService(roomRepo)
	scheduleSvc := scheduleuc.NewService(roomRepo, scheduleRepo)
	slotSvc := slotuc.NewService(roomRepo, slotRepo)
	bookingSvc := bookinguc.NewService(txm, bookingRepo, slotRepo)
	gen := slotgen.NewGenerator(roomRepo, scheduleRepo, slotRepo)

	admin := domain.User{ID: adminID, Role: domain.RoleAdmin, Email: ""}
	regular := domain.User{ID: userID, Role: domain.RoleUser, Email: ""}

	room, err := roomSvc.CreateRoom(ctx, admin, usecase.RoomCreateInput{Name: "integration-room-2"})
	require.NoError(t, err)

	nextDay := time.Now().UTC().Add(48 * time.Hour)
	listDate := time.Date(nextDay.Year(), nextDay.Month(), nextDay.Day(), 0, 0, 0, 0, time.UTC)
	dow := isoWeekdayUTC(listDate)

	_, err = scheduleSvc.CreateSchedule(ctx, admin, room.ID, usecase.CreateScheduleInput{
		DaysOfWeek: []int{dow},
		StartTime:  "10:00",
		EndTime:    "11:00",
	})
	require.NoError(t, err)
	require.NoError(t, gen.Generate(ctx))

	slots, err := slotSvc.ListAvailableSlots(ctx, regular, room.ID, listDate)
	require.NoError(t, err)
	require.NotEmpty(t, slots)
	require.True(t, slots[0].StartTime.After(time.Now().UTC()), "slot start must be in the future so CreateBooking does not flake on time")
	slotID := slots[0].ID

	_, err = bookingSvc.CreateBooking(ctx, regular, slotID, false)
	require.NoError(t, err)

	_, err = bookingSvc.CreateBooking(ctx, regular, slotID, false)
	require.Error(t, err)
	de, ok := domain.AsDomainError(err)
	require.True(t, ok)
	require.Equal(t, domain.ErrorSlotAlreadyBooked, de.Code)
}

func TestIntegration_WaitlistFlowCancelMarksEntryNotified(t *testing.T) {
	ctx := context.Background()
	pool := setupPool(t)

	roomRepo := postgres.NewRoomRepo(pool)
	scheduleRepo := postgres.NewScheduleRepo(pool)
	slotRepo := postgres.NewSlotRepo(pool)
	bookingRepo := postgres.NewBookingRepo(pool)
	waitlistRepo := postgres.NewWaitlistRepo(pool)
	txm := postgres.NewTxManager(pool)

	roomSvc := roomuc.NewService(roomRepo)
	scheduleSvc := scheduleuc.NewService(roomRepo, scheduleRepo)
	slotSvc := slotuc.NewService(roomRepo, slotRepo)
	bookingSvc := bookinguc.NewService(txm, bookingRepo, slotRepo).WithWaitlistRepository(waitlistRepo)
	waitlistSvc := waitlistuc.NewService(txm, waitlistRepo, slotRepo, bookingRepo)
	gen := slotgen.NewGenerator(roomRepo, scheduleRepo, slotRepo)

	admin := domain.User{ID: adminID, Role: domain.RoleAdmin, Email: ""}
	userA := domain.User{ID: userID, Role: domain.RoleUser, Email: ""}
	userBID := uuid.New()
	insertWaitlistUsers(ctx, t, pool, []uuid.UUID{userBID})
	userB := domain.User{ID: userBID, Role: domain.RoleUser, Email: ""}

	room, err := roomSvc.CreateRoom(ctx, admin, usecase.RoomCreateInput{Name: "integration-room-waitlist-flow"})
	require.NoError(t, err)

	nextDay := time.Now().UTC().Add(48 * time.Hour)
	listDate := time.Date(nextDay.Year(), nextDay.Month(), nextDay.Day(), 0, 0, 0, 0, time.UTC)
	dow := isoWeekdayUTC(listDate)

	_, err = scheduleSvc.CreateSchedule(ctx, admin, room.ID, usecase.CreateScheduleInput{
		DaysOfWeek: []int{dow},
		StartTime:  "10:00",
		EndTime:    "11:00",
	})
	require.NoError(t, err)
	require.NoError(t, gen.Generate(ctx))

	slots, err := slotSvc.ListAvailableSlots(ctx, userA, room.ID, listDate)
	require.NoError(t, err)
	require.NotEmpty(t, slots)
	slotID := slots[0].ID

	booking, err := bookingSvc.CreateBooking(ctx, userA, slotID, false)
	require.NoError(t, err)

	entry, err := waitlistSvc.JoinWaitlist(ctx, userB, slotID)
	require.NoError(t, err)

	_, err = bookingSvc.CancelBooking(ctx, userA, booking.ID)
	require.NoError(t, err)

	var status string
	var notifiedAt pgtype.Timestamptz
	err = pool.QueryRow(ctx, `
SELECT status, notified_at
FROM waitlist_entries
WHERE id = $1
`, entry.ID).Scan(&status, &notifiedAt)
	require.NoError(t, err)
	require.Equal(t, string(domain.WaitlistStatusNotified), status)
	require.True(t, notifiedAt.Valid)
}

func containsSlotID(slots []domain.Slot, id uuid.UUID) bool {
	for _, s := range slots {
		if s.ID == id {
			return true
		}
	}
	return false
}
