package slotgen

import (
	"context"
	"errors"
	"testing"
	"time"

	"booking-service/internal/domain"

	"github.com/google/uuid"
)

func TestGenerate_LastStartAfterTargetProducesNoSlots(t *testing.T) {
	t.Parallel()

	roomID := uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaa1000")
	schID := uuid.MustParse("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbb1000")

	now := time.Date(2025, 3, 24, 8, 0, 0, 0, time.UTC) // Mon
	target := now.Add(1 * time.Hour)                    // 09:00
	// lastStart already beyond target.
	lastStart := target.Add(30 * time.Minute) // 09:30

	slots := &mockSlotsRepo{
		lastByRoom: map[uuid.UUID]*time.Time{roomID: &lastStart},
	}

	g := NewGenerator(
		&mockRoomsRepo{rooms: []domain.Room{{ID: roomID}}},
		&mockSchedulesRepo{byRoom: map[uuid.UUID][]domain.Schedule{
			roomID: {{
				ID:        schID,
				RoomID:    roomID,
				DayOfWeek: 1,
				StartTime: "09:00:00",
				EndTime:   "11:00:00",
				CreatedAt: now,
			}},
		}},
		slots,
	)
	g.now = func() time.Time { return now }
	g.rollingWindow = 1 * time.Hour

	if err := g.Generate(context.Background()); err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if len(slots.batches) != 0 {
		t.Fatalf("expected no inserts, got %d flush calls", len(slots.batches))
	}
}

func TestGenerate_SlotStartsExactlyAtNowIncluded(t *testing.T) {
	t.Parallel()

	roomID := uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaa2000")
	schID := uuid.MustParse("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbb2000")

	now := time.Date(2025, 3, 24, 9, 0, 0, 0, time.UTC) // Mon, exactly slot start

	slots := &mockSlotsRepo{}
	g := NewGenerator(
		&mockRoomsRepo{rooms: []domain.Room{{ID: roomID}}},
		&mockSchedulesRepo{byRoom: map[uuid.UUID][]domain.Schedule{
			roomID: {{
				ID:        schID,
				RoomID:    roomID,
				DayOfWeek: 1,
				StartTime: "09:00:00",
				EndTime:   "10:00:00",
				CreatedAt: now,
			}},
		}},
		slots,
	)
	g.now = func() time.Time { return now }
	// target = now + 30m => slot starting exactly at target should be excluded.
	g.rollingWindow = 30 * time.Minute

	if err := g.Generate(context.Background()); err != nil {
		t.Fatalf("Generate: %v", err)
	}

	var all []domain.Slot
	for _, b := range slots.batches {
		all = append(all, b...)
	}
	if len(all) != 1 {
		t.Fatalf("want 1 slot, got %d", len(all))
	}
	if got := all[0].StartTime.UTC(); !got.Equal(now) {
		t.Fatalf("slot start: want %v got %v", now, got)
	}
}

func TestGenerate_LowerBoundBoundary_LastStart(t *testing.T) {
	t.Parallel()

	roomID := uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaa3000")
	schID := uuid.MustParse("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbb3000")

	now := time.Date(2025, 3, 24, 8, 0, 0, 0, time.UTC) // Mon
	lastStart := time.Date(2025, 3, 24, 9, 0, 0, 0, time.UTC)

	slots := &mockSlotsRepo{
		lastByRoom: map[uuid.UUID]*time.Time{roomID: &lastStart},
	}

	g := NewGenerator(
		&mockRoomsRepo{rooms: []domain.Room{{ID: roomID}}},
		&mockSchedulesRepo{byRoom: map[uuid.UUID][]domain.Schedule{
			roomID: {{
				ID:        schID,
				RoomID:    roomID,
				DayOfWeek: 1,
				StartTime: "09:00:00",
				EndTime:   "10:00:00",
				CreatedAt: now,
			}},
		}},
		slots,
	)
	g.now = func() time.Time { return now }
	g.rollingWindow = 2 * time.Hour

	if err := g.Generate(context.Background()); err != nil {
		t.Fatalf("Generate: %v", err)
	}

	var all []domain.Slot
	for _, b := range slots.batches {
		all = append(all, b...)
	}
	if len(all) != 1 {
		t.Fatalf("want 1 slot (only 09:30), got %d", len(all))
	}
	want := time.Date(2025, 3, 24, 9, 30, 0, 0, time.UTC)
	if got := all[0].StartTime.UTC(); !got.Equal(want) {
		t.Fatalf("slot start: want %v got %v", want, got)
	}
}

func TestGenerate_SchedulePresentButNoSlots_DoesNotInsertEmptyBatch(t *testing.T) {
	t.Parallel()

	roomID := uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaa4000")
	schID := uuid.MustParse("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbb4000")
	now := time.Date(2025, 3, 24, 8, 0, 0, 0, time.UTC) // Monday

	// Schedule exists, but only for Tuesday.
	// Use a very short rolling window so the day-iteration includes only "now" weekday,
	// therefore no slots can be appended and final flush must be a no-op.
	slots := &mockSlotsRepo{}
	g := NewGenerator(
		&mockRoomsRepo{rooms: []domain.Room{{ID: roomID}}},
		&mockSchedulesRepo{byRoom: map[uuid.UUID][]domain.Schedule{
			roomID: {{
				ID:        schID,
				RoomID:    roomID,
				DayOfWeek: 2, // Tue
				StartTime: "09:00:00",
				EndTime:   "10:00:00",
				CreatedAt: now,
			}},
		}},
		slots,
	)
	g.now = func() time.Time { return now }
	g.rollingWindow = 1 * time.Hour

	if err := g.Generate(context.Background()); err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if len(slots.batches) != 0 {
		t.Fatalf("expected no inserts, got %d flush calls", len(slots.batches))
	}
}

func TestGenerate_CreatedAtIsUniformAcrossAllSlots(t *testing.T) {
	t.Parallel()

	roomID := uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaa5000")
	schID := uuid.MustParse("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbb5000")
	now := time.Date(2025, 3, 24, 8, 0, 0, 0, time.UTC) // Mon

	slots := &mockSlotsRepo{}
	g := NewGenerator(
		&mockRoomsRepo{rooms: []domain.Room{{ID: roomID}}},
		&mockSchedulesRepo{byRoom: map[uuid.UUID][]domain.Schedule{
			roomID: {{
				ID:        schID,
				RoomID:    roomID,
				DayOfWeek: 1,
				StartTime: "09:00:00",
				EndTime:   "10:30:00", // slots: 09:00, 09:30, 10:00
				CreatedAt: now,
			}},
		}},
		slots,
	)
	g.now = func() time.Time { return now }
	g.rollingWindow = 4 * time.Hour

	if err := g.Generate(context.Background()); err != nil {
		t.Fatalf("Generate: %v", err)
	}

	var all []domain.Slot
	for _, b := range slots.batches {
		all = append(all, b...)
	}
	if len(all) == 0 {
		t.Fatal("expected slots")
	}
	for i := range all {
		if !all[i].CreatedAt.Equal(now) {
			t.Fatalf("slot %d CreatedAt mismatch: want %v got %v", i, now, all[i].CreatedAt)
		}
	}
}

func TestGenerate_ErrorStopsAfterFirstError(t *testing.T) {
	t.Parallel()

	wantErr := errors.New("unexpected error")
	roomID := uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaa6000")

	slots := &mockSlotsRepo{}
	rooms := &mockRoomsRepo{rooms: nil, err: wantErr}
	g := NewGenerator(
		rooms,
		&mockSchedulesRepo{byRoom: map[uuid.UUID][]domain.Schedule{
			roomID: {},
		}},
		slots,
	)
	g.now = func() time.Time { return time.Now().UTC() }

	err := g.Generate(context.Background())
	if !errors.Is(err, wantErr) {
		t.Fatalf("want %v got %v", wantErr, err)
	}
	if len(slots.batches) != 0 {
		t.Fatalf("expected no inserts after error, got %d flush calls", len(slots.batches))
	}
}
