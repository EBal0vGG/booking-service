package slotgen

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestComputeLowerBound_NoLastSlot(t *testing.T) {
	t.Parallel()
	now := time.Date(2025, 3, 25, 12, 0, 0, 0, time.UTC)
	got := computeLowerBound(now, nil)
	require.Equal(t, now, got)
}

func TestComputeLowerBound_UsesLastPlusSlotDuration(t *testing.T) {
	t.Parallel()
	now := time.Date(2025, 3, 25, 8, 0, 0, 0, time.UTC)
	last := time.Date(2025, 3, 25, 9, 0, 0, 0, time.UTC)
	got := computeLowerBound(now, &last)
	want := last.Add(SlotDuration)
	require.Equal(t, want, got)
}

func TestComputeLowerBound_NowAfterLastWindow(t *testing.T) {
	t.Parallel()
	now := time.Date(2025, 3, 25, 12, 0, 0, 0, time.UTC)
	last := time.Date(2025, 3, 25, 9, 0, 0, 0, time.UTC)
	got := computeLowerBound(now, &last)
	require.Equal(t, now, got)
}

func TestWeekdayMatches_ISOWeek(t *testing.T) {
	t.Parallel()
	// 2025-03-24 is Monday -> day 1
	day := time.Date(2025, 3, 24, 0, 0, 0, 0, time.UTC)
	require.True(t, weekdayMatches(day, 1))
	require.False(t, weekdayMatches(day, 2))
}
