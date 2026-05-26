package realtime

import (
	"fmt"
	"time"

	"github.com/google/uuid"
)

type EventType string
type EventTarget string

const (
	EventTypeSlotBooked             EventType = "slot_booked"
	EventTypeSlotReleased           EventType = "slot_released"
	EventTypeSlotAvailable          EventType = "slot_available"
	EventTypeSlotReserved           EventType = "slot_reserved"
	EventTypeSlotReservationExpired EventType = "slot_reservation_expired"
	EventTypeWaitlistSlotAvailable  EventType = "waitlist_slot_available"
	EventTypeWaitlistSlotReserved   EventType = "waitlist_slot_reserved"
	EventTypeReservationExpired     EventType = "reservation_expired"
)

const (
	EventTargetRoom EventTarget = "room"
	EventTargetUser EventTarget = "user"
)

type Event struct {
	Type            EventType   `json:"type"`
	Target          EventTarget `json:"target"`
	RoomID          string      `json:"roomId,omitempty"`
	UserID          string      `json:"userId,omitempty"`
	SlotID          string      `json:"slotId"`
	BookingID       string      `json:"bookingId,omitempty"`
	ReservationID   string      `json:"reservationId,omitempty"`
	WaitlistEntryID string      `json:"waitlistEntryId,omitempty"`
	ExpiresAt       string      `json:"expiresAt,omitempty"`
	Timestamp       string      `json:"timestamp"`
}

func NewRoomEvent(eventType EventType, roomID, slotID, bookingID uuid.UUID, now time.Time) Event {
	return Event{
		Type:      eventType,
		Target:    EventTargetRoom,
		RoomID:    roomID.String(),
		SlotID:    slotID.String(),
		BookingID: bookingID.String(),
		Timestamp: now.UTC().Format(time.RFC3339),
	}
}

func NewUserEvent(eventType EventType, roomID, slotID, userID, waitlistEntryID uuid.UUID, now time.Time) Event {
	return Event{
		Type:            eventType,
		Target:          EventTargetUser,
		RoomID:          roomID.String(),
		UserID:          userID.String(),
		SlotID:          slotID.String(),
		WaitlistEntryID: waitlistEntryID.String(),
		Timestamp:       now.UTC().Format(time.RFC3339),
	}
}

func NewReservationRoomEvent(eventType EventType, roomID, slotID, reservationID uuid.UUID, now time.Time) Event {
	return Event{
		Type:          eventType,
		Target:        EventTargetRoom,
		RoomID:        roomID.String(),
		SlotID:        slotID.String(),
		ReservationID: reservationID.String(),
		Timestamp:     now.UTC().Format(time.RFC3339),
	}
}

func NewSlotAvailableEvent(roomID, slotID uuid.UUID, now time.Time) Event {
	return Event{
		Type:      EventTypeSlotAvailable,
		Target:    EventTargetRoom,
		RoomID:    roomID.String(),
		SlotID:    slotID.String(),
		Timestamp: now.UTC().Format(time.RFC3339),
	}
}

func NewWaitlistReservationEvent(
	eventType EventType,
	roomID, slotID, userID, reservationID, waitlistEntryID uuid.UUID,
	expiresAt, now time.Time,
) Event {
	return Event{
		Type:            eventType,
		Target:          EventTargetUser,
		RoomID:          roomID.String(),
		UserID:          userID.String(),
		SlotID:          slotID.String(),
		ReservationID:   reservationID.String(),
		WaitlistEntryID: waitlistEntryID.String(),
		ExpiresAt:       expiresAt.UTC().Format(time.RFC3339),
		Timestamp:       now.UTC().Format(time.RFC3339),
	}
}

func NewReservationExpiredEvent(eventType EventType, roomID, slotID, userID, reservationID uuid.UUID, now time.Time) Event {
	return Event{
		Type:          eventType,
		Target:        EventTargetUser,
		RoomID:        roomID.String(),
		UserID:        userID.String(),
		SlotID:        slotID.String(),
		ReservationID: reservationID.String(),
		Timestamp:     now.UTC().Format(time.RFC3339),
	}
}

func (e Event) Validate() error {
	switch e.Type {
	case EventTypeSlotBooked, EventTypeSlotReleased, EventTypeSlotAvailable, EventTypeSlotReserved, EventTypeSlotReservationExpired, EventTypeWaitlistSlotAvailable, EventTypeWaitlistSlotReserved, EventTypeReservationExpired:
	default:
		return fmt.Errorf("unsupported event type: %s", e.Type)
	}
	if e.Target != EventTargetRoom && e.Target != EventTargetUser {
		return fmt.Errorf("unsupported event target: %s", e.Target)
	}
	switch e.Type {
	case EventTypeSlotBooked, EventTypeSlotReleased, EventTypeSlotAvailable, EventTypeSlotReserved, EventTypeSlotReservationExpired:
		if e.Target != EventTargetRoom {
			return fmt.Errorf("event type %s requires room target", e.Type)
		}
	case EventTypeWaitlistSlotAvailable, EventTypeWaitlistSlotReserved, EventTypeReservationExpired:
		if e.Target != EventTargetUser {
			return fmt.Errorf("event type %s requires user target", e.Type)
		}
	}
	if _, err := e.RoomUUID(); err != nil {
		return fmt.Errorf("invalid roomId: %w", err)
	}
	if e.Target == EventTargetUser {
		if _, err := e.UserUUID(); err != nil {
			return fmt.Errorf("invalid userId: %w", err)
		}
	}
	if _, err := uuid.Parse(e.SlotID); err != nil {
		return fmt.Errorf("invalid slotId: %w", err)
	}
	switch e.Type {
	case EventTypeSlotBooked, EventTypeSlotReleased:
		if _, err := uuid.Parse(e.BookingID); err != nil {
			return fmt.Errorf("invalid bookingId: %w", err)
		}
	case EventTypeSlotReserved, EventTypeSlotReservationExpired, EventTypeReservationExpired:
		if _, err := uuid.Parse(e.ReservationID); err != nil {
			return fmt.Errorf("invalid reservationId: %w", err)
		}
	}
	switch e.Type {
	case EventTypeWaitlistSlotAvailable, EventTypeWaitlistSlotReserved:
		if _, err := uuid.Parse(e.WaitlistEntryID); err != nil {
			return fmt.Errorf("invalid waitlistEntryId: %w", err)
		}
	}
	if e.Type == EventTypeWaitlistSlotReserved {
		if _, err := uuid.Parse(e.ReservationID); err != nil {
			return fmt.Errorf("invalid reservationId: %w", err)
		}
		if _, err := time.Parse(time.RFC3339, e.ExpiresAt); err != nil {
			return fmt.Errorf("invalid expiresAt: %w", err)
		}
	}
	if _, err := time.Parse(time.RFC3339, e.Timestamp); err != nil {
		return fmt.Errorf("invalid timestamp: %w", err)
	}
	return nil
}

func (e Event) RoomUUID() (uuid.UUID, error) {
	return uuid.Parse(e.RoomID)
}

func (e Event) UserUUID() (uuid.UUID, error) {
	return uuid.Parse(e.UserID)
}

func (e Event) ToServerMessage() ServerMessage {
	return ServerMessage{
		Type:            MessageType(e.Type),
		RoomID:          e.RoomID,
		UserID:          e.UserID,
		SlotID:          e.SlotID,
		BookingID:       e.BookingID,
		ReservationID:   e.ReservationID,
		WaitlistEntryID: e.WaitlistEntryID,
		ExpiresAt:       e.ExpiresAt,
		Timestamp:       e.Timestamp,
	}
}
