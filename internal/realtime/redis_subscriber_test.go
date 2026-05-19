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
	event := NewEvent(EventTypeSlotBooked, roomID, uuid.New(), uuid.New(), time.Now().UTC())
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
