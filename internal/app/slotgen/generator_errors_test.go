package slotgen

import (
	"context"
	"errors"
	"testing"
	"time"

	"booking-service/internal/domain"

	"github.com/google/uuid"
)

func TestGenerate_ReturnsRoomsRepoErrorAndStops(t *testing.T) {
	t.Parallel()

	wantErr := errors.New("rooms repo failed")
	rooms := &mockRoomsRepo{err: wantErr}
	schedules := &mockSchedulesRepo{}
	slots := &mockSlotsRepo{}

	g := NewGenerator(rooms, schedules, slots)
	g.now = func() time.Time { return time.Date(2025, 3, 24, 8, 0, 0, 0, time.UTC) }

	err := g.Generate(context.Background())
	if !errors.Is(err, wantErr) {
		t.Fatalf("want rooms error %v, got %v", wantErr, err)
	}
	if len(slots.batches) != 0 {
		t.Fatalf("expected no slot inserts, got %d", len(slots.batches))
	}
}

func TestGenerate_ReturnsSchedulesRepoError(t *testing.T) {
	t.Parallel()

	roomID := uuid.MustParse("12121212-1212-1212-1212-121212121212")
	wantErr := errors.New("schedules repo failed")
	rooms := &mockRoomsRepo{rooms: []domain.Room{{ID: roomID}}}
	schedules := &mockSchedulesRepo{err: wantErr}
	slots := &mockSlotsRepo{}

	g := NewGenerator(rooms, schedules, slots)
	g.now = func() time.Time { return time.Date(2025, 3, 24, 8, 0, 0, 0, time.UTC) }

	err := g.Generate(context.Background())
	if !errors.Is(err, wantErr) {
		t.Fatalf("want schedules error %v, got %v", wantErr, err)
	}
	if len(slots.batches) != 0 {
		t.Fatalf("expected no slot inserts, got %d", len(slots.batches))
	}
}

func TestGenerate_ReturnsGetLastSlotError(t *testing.T) {
	t.Parallel()

	roomID := uuid.MustParse("13131313-1313-1313-1313-131313131313")
	schID := uuid.MustParse("14141414-1414-1414-1414-141414141414")
	now := time.Date(2025, 3, 24, 8, 0, 0, 0, time.UTC)
	wantErr := errors.New("get last slot failed")

	rooms := &mockRoomsRepo{rooms: []domain.Room{{ID: roomID}}}
	schedules := &mockSchedulesRepo{byRoom: map[uuid.UUID][]domain.Schedule{
		roomID: {{
			ID:        schID,
			RoomID:    roomID,
			DayOfWeek: 1,
			StartTime: "09:00:00",
			EndTime:   "10:00:00",
			CreatedAt: now,
		}},
	}}
	slots := &mockSlotsRepo{err: wantErr}

	g := NewGenerator(rooms, schedules, slots)
	g.now = func() time.Time { return now }
	g.rollingWindow = 24 * time.Hour

	err := g.Generate(context.Background())
	if !errors.Is(err, wantErr) {
		t.Fatalf("want get-last-slot error %v, got %v", wantErr, err)
	}
	if len(slots.batches) != 0 {
		t.Fatalf("expected no slot inserts, got %d", len(slots.batches))
	}
}

func TestGenerate_ReturnsInsertBatchError(t *testing.T) {
	t.Parallel()

	roomID := uuid.MustParse("15151515-1515-1515-1515-151515151515")
	schID := uuid.MustParse("16161616-1616-1616-1616-161616161616")
	now := time.Date(2025, 3, 24, 8, 0, 0, 0, time.UTC)
	wantErr := errors.New("insert batch failed")

	rooms := &mockRoomsRepo{rooms: []domain.Room{{ID: roomID}}}
	schedules := &mockSchedulesRepo{byRoom: map[uuid.UUID][]domain.Schedule{
		roomID: {{
			ID:        schID,
			RoomID:    roomID,
			DayOfWeek: 1,
			StartTime: "09:00:00",
			EndTime:   "10:00:00",
			CreatedAt: now,
		}},
	}}
	slots := &mockSlotsRepo{err: wantErr}

	g := NewGenerator(rooms, schedules, slots)
	g.now = func() time.Time { return now }
	g.rollingWindow = 24 * time.Hour
	g.maxBatch = 1 // force flush on first generated slot

	err := g.Generate(context.Background())
	if !errors.Is(err, wantErr) {
		t.Fatalf("want insert-batch error %v, got %v", wantErr, err)
	}
}
