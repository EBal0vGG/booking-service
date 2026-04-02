package slotgen

import (
	"fmt"
	"strings"
	"time"

	"booking-service/internal/domain"
)

// SlotDuration is fixed by product rules (30 minutes per slot).
const SlotDuration = 30 * time.Minute

func truncateToUTCDate(t time.Time) time.Time {
	t = t.UTC()
	y, m, d := t.Date()
	return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
}

// dbWeekdayFromGo maps Go weekday to ISO 8601 numeric weekday (1=Monday .. 7=Sunday).
func dbWeekdayFromGo(w time.Weekday) int {
	if w == time.Sunday {
		return 7
	}
	return int(w)
}

func weekdayMatches(dayUTC time.Time, dayOfWeek int) bool {
	return dbWeekdayFromGo(dayUTC.Weekday()) == dayOfWeek
}

func parseTimeOfDay(tod domain.TimeOfDay) (time.Duration, error) {
	s := strings.TrimSpace(string(tod))
	if s == "" {
		return 0, fmt.Errorf("empty time of day")
	}

	var (
		parsed time.Time
		err    error
	)
	for _, layout := range []string{"15:04:05", "15:04"} {
		parsed, err = time.ParseInLocation(layout, s, time.UTC)
		if err == nil {
			h, m, sec := parsed.Clock()
			ns := parsed.Nanosecond()
			return time.Duration(h)*time.Hour + time.Duration(m)*time.Minute + time.Duration(sec)*time.Second + time.Duration(ns), nil
		}
	}

	return 0, fmt.Errorf("parse time of day %q: %w", s, err)
}

func combineDateWithTimeOfDayUTC(dayUTC time.Time, offset time.Duration) time.Time {
	dayUTC = truncateToUTCDate(dayUTC)
	y, m, d := dayUTC.Date()
	midnight := time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
	return midnight.Add(offset)
}

func maxTime(a, b time.Time) time.Time {
	if a.After(b) {
		return a
	}
	return b
}

func computeLowerBound(now time.Time, lastSlotStart *time.Time) time.Time {
	if lastSlotStart == nil {
		return now
	}
	return maxTime(now, lastSlotStart.Add(SlotDuration))
}
