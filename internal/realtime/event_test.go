package realtime

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestEventValidate(t *testing.T) {
	t.Parallel()

	valid := NewRoomEvent(EventTypeSlotBooked, uuid.New(), uuid.New(), uuid.New(), time.Now().UTC())
	require.NoError(t, valid.Validate())

	validUser := NewUserEvent(EventTypeWaitlistSlotAvailable, uuid.New(), uuid.New(), uuid.New(), uuid.New(), time.Now().UTC())
	require.NoError(t, validUser.Validate())
	validReserved := NewReservationRoomEvent(EventTypeSlotReserved, uuid.New(), uuid.New(), uuid.New(), time.Now().UTC())
	require.NoError(t, validReserved.Validate())
	validAvailable := NewSlotAvailableEvent(uuid.New(), uuid.New(), time.Now().UTC())
	require.NoError(t, validAvailable.Validate())
	validWaitlistReserved := NewWaitlistReservationEvent(
		EventTypeWaitlistSlotReserved,
		uuid.New(),
		uuid.New(),
		uuid.New(),
		uuid.New(),
		uuid.New(),
		time.Now().UTC().Add(5*time.Minute),
		time.Now().UTC(),
	)
	require.NoError(t, validWaitlistReserved.Validate())

	tests := []struct {
		name   string
		base   Event
		mutate func(e *Event)
	}{
		{
			name: "unsupported type",
			base: valid,
			mutate: func(e *Event) {
				e.Type = "other"
			},
		},
		{
			name: "invalid room id",
			base: valid,
			mutate: func(e *Event) {
				e.RoomID = "not-uuid"
			},
		},
		{
			name: "invalid slot id",
			base: valid,
			mutate: func(e *Event) {
				e.SlotID = "not-uuid"
			},
		},
		{
			name: "invalid booking id",
			base: valid,
			mutate: func(e *Event) {
				e.BookingID = "not-uuid"
			},
		},
		{
			name: "invalid timestamp",
			base: valid,
			mutate: func(e *Event) {
				e.Timestamp = "bad-time"
			},
		},
		{
			name: "unsupported target",
			base: valid,
			mutate: func(e *Event) {
				e.Target = "other"
			},
		},
		{
			name: "invalid user id on user target",
			base: validUser,
			mutate: func(e *Event) {
				e.UserID = "not-uuid"
			},
		},
		{
			name: "invalid waitlist entry id",
			base: validUser,
			mutate: func(e *Event) {
				e.WaitlistEntryID = "not-uuid"
			},
		},
		{
			name: "invalid reservation id",
			base: validReserved,
			mutate: func(e *Event) {
				e.ReservationID = "not-uuid"
			},
		},
		{
			name: "slot available requires room target",
			base: validAvailable,
			mutate: func(e *Event) {
				e.Target = EventTargetUser
				e.UserID = uuid.New().String()
			},
		},
		{
			name: "invalid expires at",
			base: validWaitlistReserved,
			mutate: func(e *Event) {
				e.ExpiresAt = "bad-time"
			},
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			event := tc.base
			tc.mutate(&event)
			require.Error(t, event.Validate())
		})
	}
}

func TestEventToServerMessage(t *testing.T) {
	t.Parallel()

	event := NewRoomEvent(EventTypeSlotReleased, uuid.New(), uuid.New(), uuid.New(), time.Now().UTC())
	msg := event.ToServerMessage()

	require.Equal(t, MessageTypeSlotReleased, msg.Type)
	require.Equal(t, event.RoomID, msg.RoomID)
	require.Equal(t, event.SlotID, msg.SlotID)
	require.Equal(t, event.BookingID, msg.BookingID)
	require.Equal(t, event.Timestamp, msg.Timestamp)
}

func TestWaitlistEventToServerMessage(t *testing.T) {
	t.Parallel()

	event := NewUserEvent(EventTypeWaitlistSlotAvailable, uuid.New(), uuid.New(), uuid.New(), uuid.New(), time.Now().UTC())
	msg := event.ToServerMessage()

	require.Equal(t, MessageTypeWaitlistSlotAvailable, msg.Type)
	require.Equal(t, event.RoomID, msg.RoomID)
	require.Equal(t, event.UserID, msg.UserID)
	require.Equal(t, event.SlotID, msg.SlotID)
	require.Equal(t, event.WaitlistEntryID, msg.WaitlistEntryID)
}

func TestWaitlistReservedEventToServerMessage(t *testing.T) {
	t.Parallel()

	event := NewWaitlistReservationEvent(
		EventTypeWaitlistSlotReserved,
		uuid.New(),
		uuid.New(),
		uuid.New(),
		uuid.New(),
		uuid.New(),
		time.Now().UTC().Add(5*time.Minute),
		time.Now().UTC(),
	)
	msg := event.ToServerMessage()

	require.Equal(t, MessageTypeWaitlistSlotReserved, msg.Type)
	require.Equal(t, event.ReservationID, msg.ReservationID)
	require.Equal(t, event.ExpiresAt, msg.ExpiresAt)
	require.Equal(t, event.WaitlistEntryID, msg.WaitlistEntryID)
}
