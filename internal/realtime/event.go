package realtime

import (
	"fmt"
	"time"

	"github.com/google/uuid"
)

type EventType string

const (
	EventTypeSlotBooked   EventType = "slot_booked"
	EventTypeSlotReleased EventType = "slot_released"
)

type Event struct {
	Type      EventType `json:"type"`
	RoomID    string    `json:"roomId"`
	SlotID    string    `json:"slotId"`
	BookingID string    `json:"bookingId"`
	Timestamp string    `json:"timestamp"`
}

func NewEvent(eventType EventType, roomID, slotID, bookingID uuid.UUID, now time.Time) Event {
	return Event{
		Type:      eventType,
		RoomID:    roomID.String(),
		SlotID:    slotID.String(),
		BookingID: bookingID.String(),
		Timestamp: now.UTC().Format(time.RFC3339),
	}
}

func (e Event) Validate() error {
	if e.Type != EventTypeSlotBooked && e.Type != EventTypeSlotReleased {
		return fmt.Errorf("unsupported event type: %s", e.Type)
	}
	if _, err := e.RoomUUID(); err != nil {
		return fmt.Errorf("invalid roomId: %w", err)
	}
	if _, err := uuid.Parse(e.SlotID); err != nil {
		return fmt.Errorf("invalid slotId: %w", err)
	}
	if _, err := uuid.Parse(e.BookingID); err != nil {
		return fmt.Errorf("invalid bookingId: %w", err)
	}
	if _, err := time.Parse(time.RFC3339, e.Timestamp); err != nil {
		return fmt.Errorf("invalid timestamp: %w", err)
	}
	return nil
}

func (e Event) RoomUUID() (uuid.UUID, error) {
	return uuid.Parse(e.RoomID)
}

func (e Event) ToServerMessage() ServerMessage {
	return ServerMessage{
		Type:      MessageType(e.Type),
		RoomID:    e.RoomID,
		SlotID:    e.SlotID,
		BookingID: e.BookingID,
		Timestamp: e.Timestamp,
	}
}
