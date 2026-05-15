package realtime

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestEventValidate(t *testing.T) {
	t.Parallel()

	valid := NewEvent(EventTypeSlotBooked, uuid.New(), uuid.New(), uuid.New(), time.Now().UTC())
	require.NoError(t, valid.Validate())

	tests := []struct {
		name   string
		mutate func(e *Event)
	}{
		{
			name: "unsupported type",
			mutate: func(e *Event) {
				e.Type = "other"
			},
		},
		{
			name: "invalid room id",
			mutate: func(e *Event) {
				e.RoomID = "not-uuid"
			},
		},
		{
			name: "invalid slot id",
			mutate: func(e *Event) {
				e.SlotID = "not-uuid"
			},
		},
		{
			name: "invalid booking id",
			mutate: func(e *Event) {
				e.BookingID = "not-uuid"
			},
		},
		{
			name: "invalid timestamp",
			mutate: func(e *Event) {
				e.Timestamp = "bad-time"
			},
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			event := valid
			tc.mutate(&event)
			require.Error(t, event.Validate())
		})
	}
}

func TestEventToServerMessage(t *testing.T) {
	t.Parallel()

	event := NewEvent(EventTypeSlotReleased, uuid.New(), uuid.New(), uuid.New(), time.Now().UTC())
	msg := event.ToServerMessage()

	require.Equal(t, MessageTypeSlotReleased, msg.Type)
	require.Equal(t, event.RoomID, msg.RoomID)
	require.Equal(t, event.SlotID, msg.SlotID)
	require.Equal(t, event.BookingID, msg.BookingID)
	require.Equal(t, event.Timestamp, msg.Timestamp)
}
