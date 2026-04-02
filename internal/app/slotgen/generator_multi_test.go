package slotgen

import (
	"context"
	"strings"
	"testing"
	"time"

	"booking-service/internal/domain"

	"github.com/google/uuid"
)

func TestGenerate_MultipleRoomsDifferentSchedules(t *testing.T) {
	t.Parallel()

	room1 := uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaa0001")
	room2 := uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaa0002")
	sch1 := uuid.MustParse("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbb0001")
	sch2 := uuid.MustParse("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbb0002")

	now := time.Date(2025, 3, 24, 8, 0, 0, 0, time.UTC) // Monday

	slots := &mockSlotsRepo{}
	g := NewGenerator(
		&mockRoomsRepo{rooms: []domain.Room{{ID: room1, Name: "r1"}, {ID: room2, Name: "r2"}}},
		&mockSchedulesRepo{byRoom: map[uuid.UUID][]domain.Schedule{
			room1: {{
				ID:        sch1,
				RoomID:    room1,
				DayOfWeek: 1,
				StartTime: "09:00:00",
				EndTime:   "10:00:00",
				CreatedAt: now,
			}},
			room2: {{
				ID:        sch2,
				RoomID:    room2,
				DayOfWeek: 1,
				StartTime: "15:00:00",
				EndTime:   "16:00:00",
				CreatedAt: now,
			}},
		}},
		slots,
	)
	g.now = func() time.Time { return now }
	g.rollingWindow = 24 * time.Hour

	if err := g.Generate(context.Background()); err != nil {
		t.Fatalf("Generate: %v", err)
	}

	var (
		byRoom     = map[uuid.UUID][]domain.Slot{}
		flattened  []domain.Slot
		expected1  = 2
		expected2  = 2
	)
	for _, b := range slots.batches {
		flattened = append(flattened, b...)
		for _, s := range b {
			byRoom[s.RoomID] = append(byRoom[s.RoomID], s)
		}
	}

	if len(flattened) != expected1+expected2 {
		t.Fatalf("want %d slots total, got %d", expected1+expected2, len(flattened))
	}
	if len(byRoom[room1]) != expected1 {
		t.Fatalf("room1: want %d slots, got %d", expected1, len(byRoom[room1]))
	}
	if len(byRoom[room2]) != expected2 {
		t.Fatalf("room2: want %d slots, got %d", expected2, len(byRoom[room2]))
	}
	for _, s := range flattened {
		switch s.RoomID {
		case room1:
			// Starts inside 09:00-10:00 schedule.
			if s.StartTime.UTC().Hour() != 9 {
				t.Fatalf("slot room1 has unexpected start %v", s.StartTime)
			}
		case room2:
			if s.StartTime.UTC().Hour() != 15 {
				t.Fatalf("slot room2 has unexpected start %v", s.StartTime)
			}
		default:
			t.Fatalf("unexpected RoomID in slots: %v", s.RoomID)
		}
	}
}

func TestGenerate_MultipleSchedulesSameRoomSameDay(t *testing.T) {
	t.Parallel()

	room := uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaa0010")
	sch1 := uuid.MustParse("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbb0011")
	sch2 := uuid.MustParse("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbb0012")

	now := time.Date(2025, 3, 24, 8, 0, 0, 0, time.UTC) // Monday

	slots := &mockSlotsRepo{}
	g := NewGenerator(
		&mockRoomsRepo{rooms: []domain.Room{{ID: room, Name: "r"}}},
		&mockSchedulesRepo{byRoom: map[uuid.UUID][]domain.Schedule{
			room: {
				// Intentionally out-of-order to verify sorting doesn't break results.
				{
					ID:        sch2,
					RoomID:    room,
					DayOfWeek: 1,
					StartTime: "14:00:00",
					EndTime:   "15:00:00",
					CreatedAt: now,
				},
				{
					ID:        sch1,
					RoomID:    room,
					DayOfWeek: 1,
					StartTime: "09:00:00",
					EndTime:   "10:00:00",
					CreatedAt: now,
				},
			},
		}},
		slots,
	)
	g.now = func() time.Time { return now }
	g.rollingWindow = 24 * time.Hour

	if err := g.Generate(context.Background()); err != nil {
		t.Fatalf("Generate: %v", err)
	}

	var flattened []domain.Slot
	for _, b := range slots.batches {
		flattened = append(flattened, b...)
	}

	if len(flattened) != 4 {
		t.Fatalf("want 4 slots, got %d", len(flattened))
	}

	// Expect sorted by schedule start time: 09:00, 09:30, 14:00, 14:30.
	wantStarts := []string{
		"2025-03-24T09:00:00Z",
		"2025-03-24T09:30:00Z",
		"2025-03-24T14:00:00Z",
		"2025-03-24T14:30:00Z",
	}
	for i, w := range wantStarts {
		got := flattened[i].StartTime.UTC().Format(time.RFC3339)
		if got != w {
			t.Fatalf("slot %d start: want %s got %s", i, w, got)
		}
	}
	for _, s := range flattened {
		if s.RoomID != room {
			t.Fatalf("RoomID mixed: want %v got %v", room, s.RoomID)
		}
	}
}

func TestGenerate_InvalidTimeOfDayStartOrEndTimeFails(t *testing.T) {
	t.Parallel()

	room := uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaa0020")
	sch := uuid.MustParse("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbb0020")

	now := time.Date(2025, 3, 24, 8, 0, 0, 0, time.UTC) // Monday

	tests := []struct {
		name       string
		startTime  domain.TimeOfDay
		endTime    domain.TimeOfDay
	}{
		{name: "invalid_start", startTime: domain.TimeOfDay("not-a-time"), endTime: "10:00:00"},
		{name: "invalid_end", startTime: "09:00:00", endTime: domain.TimeOfDay("bad")},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// fresh slots each subtest
			slots := &mockSlotsRepo{}
			g := NewGenerator(
				&mockRoomsRepo{rooms: []domain.Room{{ID: room, Name: "r"}}},
				&mockSchedulesRepo{byRoom: map[uuid.UUID][]domain.Schedule{
					room: {{
						ID:        sch,
						RoomID:    room,
						DayOfWeek: 1,
						StartTime: tc.startTime,
						EndTime:   tc.endTime,
						CreatedAt: now,
					}},
				}},
				slots,
			)
			g.now = func() time.Time { return now }
			g.rollingWindow = 24 * time.Hour

			err := g.Generate(context.Background())
			if err == nil {
				t.Fatalf("expected error, got nil")
			}
			if !strings.Contains(err.Error(), "parse time of day") {
				t.Fatalf("expected parse time of day error, got: %v", err)
			}
			if len(slots.batches) != 0 {
				t.Fatalf("expected no inserts on invalid schedule, got %d calls", len(slots.batches))
			}
		})
	}
}

func TestGenerate_EndTimeLessOrEqualStartTimeProducesNoSlots(t *testing.T) {
	t.Parallel()

	room := uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaa0030")
	sch := uuid.MustParse("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbb0030")

	now := time.Date(2025, 3, 24, 8, 0, 0, 0, time.UTC) // Monday
	slots := &mockSlotsRepo{}

	g := NewGenerator(
		&mockRoomsRepo{rooms: []domain.Room{{ID: room, Name: "r"}}},
		&mockSchedulesRepo{byRoom: map[uuid.UUID][]domain.Schedule{
			room: {{
				ID:        sch,
				RoomID:    room,
				DayOfWeek: 1,
				StartTime: domain.TimeOfDay("10:00:00"),
				EndTime:   domain.TimeOfDay("09:30:00"),
				CreatedAt: now,
			}},
		}},
		slots,
	)
	g.now = func() time.Time { return now }
	g.rollingWindow = 24 * time.Hour

	if err := g.Generate(context.Background()); err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if len(slots.batches) != 0 {
		t.Fatalf("expected no inserts when EndTime <= StartTime, got %d flush calls", len(slots.batches))
	}
}

