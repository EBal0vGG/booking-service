package slotgen

import (
	"context"
	"sort"
	"time"

	"booking-service/internal/domain"

	"github.com/google/uuid"
)

const (
	DefaultRollingWindow = 14 * 24 * time.Hour
	// DefaultMaxBatch caps rows per InsertBatchIgnoreConflicts call when schedule yields many slots.
	DefaultMaxBatch = 500
)

type Generator struct {
	rooms     RoomRepository
	schedules ScheduleRepository
	slots     SlotRepository

	now           func() time.Time
	rollingWindow time.Duration
	maxBatch      int
}

func NewGenerator(
	rooms RoomRepository,
	schedules ScheduleRepository,
	slots SlotRepository,
) *Generator {
	return &Generator{
		rooms:         rooms,
		schedules:     schedules,
		slots:         slots,
		now:           time.Now().UTC,
		rollingWindow: DefaultRollingWindow,
		maxBatch:      DefaultMaxBatch,
	}
}

func (g *Generator) Generate(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	now := g.now().UTC()
	target := now.Add(g.rollingWindow)

	rooms, err := g.rooms.ListWithSchedule(ctx)
	if err != nil {
		return err
	}
	if len(rooms) == 0 {
		return nil
	}

	roomIDs := make([]uuid.UUID, len(rooms))
	for i := range rooms {
		roomIDs[i] = rooms[i].ID
	}

	schedByRoom, err := g.schedules.ListByRoomIDs(ctx, roomIDs)
	if err != nil {
		return err
	}

	for _, room := range rooms {
		if err := ctx.Err(); err != nil {
			return err
		}

		schedules := schedByRoom[room.ID]
		if len(schedules) == 0 {
			continue
		}

		createdAt := now
		if err := g.generateForRoom(ctx, room, schedules, now, target, createdAt); err != nil {
			return err
		}
	}

	return nil
}

func (g *Generator) generateForRoom(ctx context.Context, room domain.Room, schedules []domain.Schedule, now, target, createdAt time.Time) error {
	lastStart, err := g.slots.GetLastSlotStartByRoomID(ctx, room.ID)
	if err != nil {
		return err
	}

	lowerBound := computeLowerBound(now, lastStart)

	sort.Slice(schedules, func(i, j int) bool {
		a, b := schedules[i], schedules[j]
		if a.DayOfWeek != b.DayOfWeek {
			return a.DayOfWeek < b.DayOfWeek
		}
		da, errA := parseTimeOfDay(a.StartTime)
		db, errB := parseTimeOfDay(b.StartTime)
		if errA != nil || errB != nil {
			return a.ID.String() < b.ID.String()
		}
		if da != db {
			return da < db
		}
		return a.ID.String() < b.ID.String()
	})

	startDay := truncateToUTCDate(lowerBound)
	endDay := truncateToUTCDate(target)

	batch := make([]domain.Slot, 0, g.maxBatch)

	for day := startDay; !day.After(endDay); day = day.AddDate(0, 0, 1) {
		if err := ctx.Err(); err != nil {
			return err
		}

		for _, sch := range schedules {
			if !weekdayMatches(day, sch.DayOfWeek) {
				continue
			}

			if err := g.generateForSchedule(ctx, &batch, room.ID, day, sch, lowerBound, target, createdAt); err != nil {
				return err
			}
		}
	}

	return g.flushBatch(ctx, &batch)
}

func (g *Generator) generateForSchedule(
	ctx context.Context,
	batch *[]domain.Slot,
	roomID uuid.UUID,
	dayUTC time.Time,
	sch domain.Schedule,
	lowerBound, target time.Time,
	createdAt time.Time,
) error {
	startDur, err := parseTimeOfDay(sch.StartTime)
	if err != nil {
		return err
	}
	endDur, err := parseTimeOfDay(sch.EndTime)
	if err != nil {
		return err
	}

	startOfDay := combineDateWithTimeOfDayUTC(dayUTC, startDur)
	endOfDay := combineDateWithTimeOfDayUTC(dayUTC, endDur)

	// Schedule bounds may be any TIME values; we emit only full SlotDuration intervals fully inside [startOfDay, endOfDay).
	// Any remainder shorter than SlotDuration at the end of the window is intentionally dropped.
	for slotStart := startOfDay; !slotStart.Add(SlotDuration).After(endOfDay); slotStart = slotStart.Add(SlotDuration) {
		if err := ctx.Err(); err != nil {
			return err
		}

		if slotStart.Before(lowerBound) {
			continue
		}

		// Rolling window is [now, target) by slot start: include starts strictly before target.
		if !slotStart.Before(target) {
			break
		}

		slotEnd := slotStart.Add(SlotDuration)

		slot := g.buildSlot(roomID, slotStart, slotEnd, createdAt)
		if err := g.appendSlot(ctx, batch, slot); err != nil {
			return err
		}
	}

	return nil
}

func (g *Generator) buildSlot(roomID uuid.UUID, start, end, createdAt time.Time) domain.Slot {
	return domain.Slot{
		ID:        stableSlotID(roomID, start),
		RoomID:    roomID,
		StartTime: start.UTC(),
		EndTime:   end.UTC(),
		CreatedAt: createdAt.UTC(),
	}
}

func (g *Generator) appendSlot(ctx context.Context, batch *[]domain.Slot, slot domain.Slot) error {
	*batch = append(*batch, slot)
	if len(*batch) < g.maxBatch {
		return nil
	}
	return g.flushBatch(ctx, batch)
}

func (g *Generator) flushBatch(ctx context.Context, batch *[]domain.Slot) error {
	if len(*batch) == 0 {
		return nil
	}
	if err := g.slots.InsertBatchIgnoreConflicts(ctx, *batch); err != nil {
		return err
	}
	*batch = (*batch)[:0]
	return nil
}
