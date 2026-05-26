package realtime

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestRedisSubscriberHandleMessage_ValidEventBroadcastsToRoom(t *testing.T) {
	t.Parallel()

	hub := NewHub()
	client := &Client{send: make(chan outboundMessage, 1)}
	roomID := uuid.New()
	hub.Subscribe(roomID, client)

	subscriber := &RedisSubscriber{hub: hub}
	event := NewRoomEvent(EventTypeSlotBooked, roomID, uuid.New(), uuid.New(), time.Now().UTC())
	payload, err := json.Marshal(event)
	require.NoError(t, err)

	err = subscriber.handleMessage(string(payload))
	require.NoError(t, err)

	select {
	case outbound := <-client.send:
		var msg ServerMessage
		require.NoError(t, json.Unmarshal(outbound.payload, &msg))
		require.Equal(t, MessageTypeSlotBooked, msg.Type)
		require.Equal(t, event.RoomID, msg.RoomID)
		require.Equal(t, event.SlotID, msg.SlotID)
		require.Equal(t, event.BookingID, msg.BookingID)
		require.Equal(t, event.Timestamp, msg.Timestamp)
	default:
		t.Fatal("expected broadcast payload for subscribed client")
	}
}

func TestRedisSubscriberHandleMessage_SlotAvailableBroadcastsToRoom(t *testing.T) {
	t.Parallel()

	hub := NewHub()
	client := &Client{send: make(chan outboundMessage, 1)}
	roomID := uuid.New()
	hub.Subscribe(roomID, client)

	subscriber := &RedisSubscriber{hub: hub}
	event := NewSlotAvailableEvent(roomID, uuid.New(), time.Now().UTC())
	payload, err := json.Marshal(event)
	require.NoError(t, err)

	err = subscriber.handleMessage(string(payload))
	require.NoError(t, err)

	select {
	case outbound := <-client.send:
		var msg ServerMessage
		require.NoError(t, json.Unmarshal(outbound.payload, &msg))
		require.Equal(t, MessageTypeSlotAvailable, msg.Type)
		require.Equal(t, event.RoomID, msg.RoomID)
		require.Equal(t, event.SlotID, msg.SlotID)
	default:
		t.Fatal("expected broadcast payload for slot_available event")
	}
}

func TestRedisSubscriberHandleMessage_UserTargetSendsToUser(t *testing.T) {
	t.Parallel()

	hub := NewHub()
	userID := uuid.New()
	client := &Client{
		send: make(chan outboundMessage, 1),
	}
	hub.RegisterUser(userID, client)

	subscriber := &RedisSubscriber{hub: hub}
	event := NewUserEvent(EventTypeWaitlistSlotAvailable, uuid.New(), uuid.New(), userID, uuid.New(), time.Now().UTC())
	payload, err := json.Marshal(event)
	require.NoError(t, err)

	err = subscriber.handleMessage(string(payload))
	require.NoError(t, err)

	select {
	case outbound := <-client.send:
		var msg ServerMessage
		require.NoError(t, json.Unmarshal(outbound.payload, &msg))
		require.Equal(t, MessageTypeWaitlistSlotAvailable, msg.Type)
		require.Equal(t, event.UserID, msg.UserID)
		require.Equal(t, event.WaitlistEntryID, msg.WaitlistEntryID)
	default:
		t.Fatal("expected user-target payload for registered user client")
	}
}

func TestRedisSubscriberHandleMessage_WaitlistReservedUserTarget(t *testing.T) {
	t.Parallel()

	hub := NewHub()
	userID := uuid.New()
	client := &Client{
		send: make(chan outboundMessage, 1),
	}
	hub.RegisterUser(userID, client)

	subscriber := &RedisSubscriber{hub: hub}
	event := NewWaitlistReservationEvent(
		EventTypeWaitlistSlotReserved,
		uuid.New(),
		uuid.New(),
		userID,
		uuid.New(),
		uuid.New(),
		time.Now().UTC().Add(5*time.Minute),
		time.Now().UTC(),
	)
	payload, err := json.Marshal(event)
	require.NoError(t, err)

	err = subscriber.handleMessage(string(payload))
	require.NoError(t, err)

	select {
	case outbound := <-client.send:
		var msg ServerMessage
		require.NoError(t, json.Unmarshal(outbound.payload, &msg))
		require.Equal(t, MessageTypeWaitlistSlotReserved, msg.Type)
		require.Equal(t, event.UserID, msg.UserID)
		require.Equal(t, event.WaitlistEntryID, msg.WaitlistEntryID)
		require.Equal(t, event.ReservationID, msg.ReservationID)
		require.Equal(t, event.ExpiresAt, msg.ExpiresAt)
	default:
		t.Fatal("expected user-target reserved payload for registered user client")
	}
}

func TestRedisSubscriberHandleMessage_InvalidJSON(t *testing.T) {
	t.Parallel()

	subscriber := &RedisSubscriber{hub: NewHub()}
	err := subscriber.handleMessage("{bad-json")
	require.Error(t, err)
	require.ErrorContains(t, err, "unmarshal event")
}

func TestRedisSubscriberHandleMessage_InvalidEvent(t *testing.T) {
	t.Parallel()

	hub := NewHub()
	client := &Client{send: make(chan outboundMessage, 1)}
	hub.Subscribe(uuid.New(), client)

	event := Event{
		Type:      EventTypeSlotReleased,
		Target:    EventTargetRoom,
		RoomID:    "invalid-room-id",
		SlotID:    uuid.New().String(),
		BookingID: uuid.New().String(),
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}
	payload, err := json.Marshal(event)
	require.NoError(t, err)

	subscriber := &RedisSubscriber{hub: hub}
	err = subscriber.handleMessage(string(payload))
	require.Error(t, err)
	require.ErrorContains(t, err, "invalid event")

	select {
	case <-client.send:
		t.Fatal("did not expect broadcast for invalid event")
	default:
	}
}
