package slotgen

import (
	"context"
	"errors"
	"testing"
	"time"

	"booking-service/internal/domain"

	"github.com/google/uuid"
)

type mockRoomsRepo struct {
	rooms []domain.Room
	err   error
}

func (m *mockRoomsRepo) ListWithSchedule(_ context.Context) ([]domain.Room, error) {
	return m.rooms, m.err
}

type mockSchedulesRepo struct {
	byRoom map[uuid.UUID][]domain.Schedule
	err    error
}

func (m *mockSchedulesRepo) ListByRoomIDs(_ context.Context, _ []uuid.UUID) (map[uuid.UUID][]domain.Schedule, error) {
	if m.byRoom == nil {
		return map[uuid.UUID][]domain.Schedule{}, m.err
	}
	return m.byRoom, m.err
}

type mockSlotsRepo struct {
	lastByRoom map[uuid.UUID]*time.Time
	batches    [][]domain.Slot
	err        error
}

func (m *mockSlotsRepo) GetLastSlotStartByRoomID(_ context.Context, roomID uuid.UUID) (*time.Time, error) {
	if m.err != nil {
		return nil, m.err
	}
	if m.lastByRoom == nil {
		return nil, nil
	}
	return m.lastByRoom[roomID], nil
}

func (m *mockSlotsRepo) InsertBatchIgnoreConflicts(_ context.Context, slots []domain.Slot) error {
	if m.err != nil {
		return m.err
	}
	cp := append([]domain.Slot(nil), slots...)
	m.batches = append(m.batches, cp)
	return nil
}

func TestGenerate_EmptyRooms(t *testing.T) {
	t.Parallel()

	slotsRepo := &mockSlotsRepo{}
	g := NewGenerator(&mockRoomsRepo{rooms: nil}, &mockSchedulesRepo{}, slotsRepo)
	g.now = func() time.Time { return time.Date(2025, 3, 24, 10, 0, 0, 0, time.UTC) }

	if err := g.Generate(context.Background()); err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if len(slotsRepo.batches) != 0 {
		t.Fatalf("want no flush calls, got %d", len(slotsRepo.batches))
	}
}

func TestGenerate_RoomWithoutScheduleSkipped(t *testing.T) {
	t.Parallel()

	roomID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	slotsRepo := &mockSlotsRepo{}
	g := NewGenerator(
		&mockRoomsRepo{rooms: []domain.Room{{ID: roomID, Name: "r"}}},
		&mockSchedulesRepo{byRoom: map[uuid.UUID][]domain.Schedule{}},
		slotsRepo,
	)
	g.now = func() time.Time { return time.Date(2025, 3, 24, 10, 0, 0, 0, time.UTC) }

	if err := g.Generate(context.Background()); err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if len(slotsRepo.batches) != 0 {
		t.Fatalf("InsertBatchIgnoreConflicts should not be called, got %d calls", len(slotsRepo.batches))
	}
}

func TestGenerate_OneScheduleTwoSlots(t *testing.T) {
	t.Parallel()

	roomID := uuid.MustParse("22222222-2222-2222-2222-222222222222")
	schID := uuid.MustParse("33333333-3333-3333-3333-333333333333")
	now := time.Date(2025, 3, 24, 8, 0, 0, 0, time.UTC) // Monday

	slots := &mockSlotsRepo{}
	g := NewGenerator(
		&mockRoomsRepo{rooms: []domain.Room{{ID: roomID, Name: "r"}}},
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
	g.rollingWindow = 24 * time.Hour

	if err := g.Generate(context.Background()); err != nil {
		t.Fatalf("Generate: %v", err)
	}

	var all []domain.Slot
	for _, b := range slots.batches {
		all = append(all, b...)
	}
	if len(all) != 2 {
		t.Fatalf("want 2 slots, got %d", len(all))
	}
	wantStarts := []string{
		"2025-03-24T09:00:00Z",
		"2025-03-24T09:30:00Z",
	}
	wantEnds := []string{
		"2025-03-24T09:30:00Z",
		"2025-03-24T10:00:00Z",
	}
	for i, w := range wantStarts {
		if got := all[i].StartTime.UTC().Format(time.RFC3339); got != w {
			t.Fatalf("slot %d start: want %s got %s", i, w, got)
		}
		if got := all[i].EndTime.UTC().Format(time.RFC3339); got != wantEnds[i] {
			t.Fatalf("slot %d end: want %s got %s", i, wantEnds[i], got)
		}
		if all[i].RoomID != roomID {
			t.Fatalf("slot %d room_id: want %s got %s", i, roomID, all[i].RoomID)
		}
		wantID := stableSlotID(roomID, all[i].StartTime)
		if all[i].ID != wantID {
			t.Fatalf("slot %d id: want %s got %s", i, wantID, all[i].ID)
		}
	}
	for _, s := range all {
		if !s.CreatedAt.Equal(now) {
			t.Fatalf("CreatedAt want %v got %v", now, s.CreatedAt)
		}
	}
}

func TestGenerate_OnlyMatchingWeekday(t *testing.T) {
	t.Parallel()

	roomID := uuid.MustParse("44444444-4444-4444-4444-444444444444")
	schID := uuid.MustParse("55555555-5555-5555-5555-555555555555")
	// Monday 2025-03-24; Wednesday is 2025-03-26 (day_of_week 3).
	now := time.Date(2025, 3, 24, 8, 0, 0, 0, time.UTC)

	slots := &mockSlotsRepo{}
	g := NewGenerator(
		&mockRoomsRepo{rooms: []domain.Room{{ID: roomID}}},
		&mockSchedulesRepo{byRoom: map[uuid.UUID][]domain.Schedule{
			roomID: {{
				ID:        schID,
				RoomID:    roomID,
				DayOfWeek: 3,
				StartTime: "09:00:00",
				EndTime:   "09:30:00",
				CreatedAt: now,
			}},
		}},
		slots,
	)
	g.now = func() time.Time { return now }
	g.rollingWindow = 7 * 24 * time.Hour

	if err := g.Generate(context.Background()); err != nil {
		t.Fatalf("Generate: %v", err)
	}

	var all []domain.Slot
	for _, b := range slots.batches {
		all = append(all, b...)
	}
	if len(all) != 1 {
		t.Fatalf("want 1 slot on Wednesday, got %d", len(all))
	}
	if got := all[0].StartTime.UTC(); got.Weekday() != time.Wednesday {
		t.Fatalf("want Wednesday, got %v", got.Weekday())
	}
}

func TestGenerate_LastSlotRaisesLowerBound(t *testing.T) {
	t.Parallel()

	roomID := uuid.MustParse("66666666-6666-6666-6666-666666666666")
	schID := uuid.MustParse("77777777-7777-7777-7777-777777777777")
	now := time.Date(2025, 3, 24, 8, 0, 0, 0, time.UTC)
	last := time.Date(2025, 3, 24, 9, 0, 0, 0, time.UTC)

	slots := &mockSlotsRepo{
		lastByRoom: map[uuid.UUID]*time.Time{roomID: &last},
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
	g.rollingWindow = 24 * time.Hour

	if err := g.Generate(context.Background()); err != nil {
		t.Fatalf("Generate: %v", err)
	}

	var all []domain.Slot
	for _, b := range slots.batches {
		all = append(all, b...)
	}
	if len(all) != 1 {
		t.Fatalf("want 1 slot after last+30m, got %d", len(all))
	}
	want := time.Date(2025, 3, 24, 9, 30, 0, 0, time.UTC)
	if !all[0].StartTime.Equal(want) {
		t.Fatalf("start want %v got %v", want, all[0].StartTime)
	}
}

func TestGenerate_NoPastSlots(t *testing.T) {
	t.Parallel()

	roomID := uuid.MustParse("88888888-8888-8888-8888-888888888888")
	schID := uuid.MustParse("99999999-9999-9999-9999-999999999999")
	now := time.Date(2025, 3, 24, 10, 15, 0, 0, time.UTC) // Monday

	slots := &mockSlotsRepo{}
	g := NewGenerator(
		&mockRoomsRepo{rooms: []domain.Room{{ID: roomID}}},
		&mockSchedulesRepo{byRoom: map[uuid.UUID][]domain.Schedule{
			roomID: {{
				ID:        schID,
				RoomID:    roomID,
				DayOfWeek: 1,
				StartTime: "09:00:00",
				EndTime:   "18:00:00",
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

	var all []domain.Slot
	for _, b := range slots.batches {
		all = append(all, b...)
	}
	for _, s := range all {
		if s.StartTime.Before(now) {
			t.Fatalf("slot in past: %v < %v", s.StartTime, now)
		}
	}
	if len(all) == 0 {
		t.Fatal("expected slots")
	}
}

func TestGenerate_BatchFlush(t *testing.T) {
	t.Parallel()

	roomID := uuid.MustParse("aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")
	schID := uuid.MustParse("bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb")
	now := time.Date(2025, 3, 24, 8, 0, 0, 0, time.UTC)

	slots := &mockSlotsRepo{}
	g := NewGenerator(
		&mockRoomsRepo{rooms: []domain.Room{{ID: roomID}}},
		&mockSchedulesRepo{byRoom: map[uuid.UUID][]domain.Schedule{
			roomID: {{
				ID:        schID,
				RoomID:    roomID,
				DayOfWeek: 1,
				StartTime: "09:00:00",
				EndTime:   "12:00:00",
				CreatedAt: now,
			}},
		}},
		slots,
	)
	g.now = func() time.Time { return now }
	g.rollingWindow = 24 * time.Hour
	g.maxBatch = 2

	if err := g.Generate(context.Background()); err != nil {
		t.Fatalf("Generate: %v", err)
	}

	// 09:00-12:00 with 30m step -> 6 slots total.
	const expectedSlots = 6
	const expectedFlushes = 3 // maxBatch=2 => 2+2+2
	if len(slots.batches) != expectedFlushes {
		t.Fatalf("want %d flushes, got %d", expectedFlushes, len(slots.batches))
	}

	var (
		total       int
		flattened   []domain.Slot
		prevStartTS *time.Time
	)
	for i, b := range slots.batches {
		if len(b) == 0 {
			t.Fatalf("flush %d is empty", i)
		}
		if len(b) > g.maxBatch {
			t.Fatalf("flush %d size exceeds maxBatch: size=%d maxBatch=%d", i, len(b), g.maxBatch)
		}
		total += len(b)
		flattened = append(flattened, b...)

		for j := range b {
			if prevStartTS != nil && b[j].StartTime.Before(*prevStartTS) {
				t.Fatalf("slot order broken at flush=%d idx=%d: %v before %v", i, j, b[j].StartTime, *prevStartTS)
			}
			curr := b[j].StartTime
			prevStartTS = &curr
		}
	}
	if total != expectedSlots {
		t.Fatalf("want %d slots inserted total, got %d", expectedSlots, total)
	}

	expectedStarts := []string{
		"2025-03-24T09:00:00Z",
		"2025-03-24T09:30:00Z",
		"2025-03-24T10:00:00Z",
		"2025-03-24T10:30:00Z",
		"2025-03-24T11:00:00Z",
		"2025-03-24T11:30:00Z",
	}
	for i, s := range flattened {
		got := s.StartTime.UTC().Format(time.RFC3339)
		if got != expectedStarts[i] {
			t.Fatalf("slot order/start mismatch at %d: want %s got %s", i, expectedStarts[i], got)
		}
	}
}

func TestStableSlotID_Deterministic(t *testing.T) {
	t.Parallel()

	roomID := uuid.MustParse("cccccccc-cccc-cccc-cccc-cccccccccccc")
	start := time.Date(2025, 3, 24, 9, 0, 0, 0, time.UTC)
	a := stableSlotID(roomID, start)
	b := stableSlotID(roomID, start)
	if a != b {
		t.Fatalf("stableSlotID not deterministic")
	}

	// Same room, different time -> different UUID.
	c := stableSlotID(roomID, start.Add(30*time.Minute))
	if a == c {
		t.Fatalf("stableSlotID should differ for different start time: %s vs %s", a, c)
	}

	// Different room, same time -> different UUID.
	otherRoomID := uuid.MustParse("abababab-abab-abab-abab-abababababab")
	d := stableSlotID(otherRoomID, start)
	if a == d {
		t.Fatalf("stableSlotID should differ for different room: %s vs %s", a, d)
	}

	// Local time vs UTC equivalent instant -> same UUID.
	loc := time.FixedZone("UTC+3", 3*3600)
	startInLoc := start.In(loc) // same instant, different location
	e := stableSlotID(roomID, startInLoc)
	if a != e {
		t.Fatalf("stableSlotID should match for same instant across locations: want %s got %s", a, e)
	}
}

func TestGenerate_RollingWindowByStart(t *testing.T) {
	t.Parallel()

	roomID := uuid.MustParse("dddddddd-dddd-dddd-dddd-dddddddddddd")
	schID := uuid.MustParse("eeeeeeee-eeee-eeee-eeee-eeeeeeeeeeee")
	// Window is [now, target) by slot start: include starts strictly before target.
	// 09:00 and 09:30 are included; 10:00 start is excluded (not Before target).
	now := time.Date(2025, 3, 24, 8, 0, 0, 0, time.UTC)
	target := time.Date(2025, 3, 24, 10, 0, 0, 0, time.UTC)

	slots := &mockSlotsRepo{}
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
	g.rollingWindow = target.Sub(now)

	if err := g.Generate(context.Background()); err != nil {
		t.Fatalf("Generate: %v", err)
	}

	var all []domain.Slot
	for _, b := range slots.batches {
		all = append(all, b...)
	}
	if len(all) != 2 {
		t.Fatalf("want 2 slots (09:00, 09:30 starts before target), got %d", len(all))
	}
	last := all[len(all)-1].StartTime.UTC()
	if !last.Equal(time.Date(2025, 3, 24, 9, 30, 0, 0, time.UTC)) {
		t.Fatalf("last start want 09:30 got %v", last)
	}
}

func TestGenerate_ContextCancelled(t *testing.T) {
	t.Parallel()

	roomID := uuid.MustParse("ffffffff-ffff-ffff-ffff-ffffffffffff")
	schID := uuid.MustParse("10101010-1010-1010-1010-101010101010")
	now := time.Date(2025, 3, 24, 8, 0, 0, 0, time.UTC)

	g := NewGenerator(
		&mockRoomsRepo{rooms: []domain.Room{{ID: roomID}}},
		&mockSchedulesRepo{byRoom: map[uuid.UUID][]domain.Schedule{
			roomID: {{
				ID:        schID,
				RoomID:    roomID,
				DayOfWeek: 1,
				StartTime: "09:00:00",
				EndTime:   "18:00:00",
				CreatedAt: now,
			}},
		}},
		&mockSlotsRepo{},
	)
	g.now = func() time.Time { return now }

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := g.Generate(ctx)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("want context.Canceled, got %v", err)
	}
}
