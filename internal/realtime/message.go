package realtime

type MessageType string

const (
	MessageTypeSubscribe   MessageType = "subscribe"
	MessageTypeUnsubscribe MessageType = "unsubscribe"

	MessageTypeSubscribed             MessageType = "subscribed"
	MessageTypeUnsubscribed           MessageType = "unsubscribed"
	MessageTypeSlotBooked             MessageType = "slot_booked"
	MessageTypeSlotReleased           MessageType = "slot_released"
	MessageTypeSlotAvailable          MessageType = "slot_available"
	MessageTypeSlotReserved           MessageType = "slot_reserved"
	MessageTypeSlotReservationExpired MessageType = "slot_reservation_expired"
	MessageTypeWaitlistSlotAvailable  MessageType = "waitlist_slot_available"
	MessageTypeWaitlistSlotReserved   MessageType = "waitlist_slot_reserved"
	MessageTypeReservationExpired     MessageType = "reservation_expired"
	MessageTypeError                  MessageType = "error"
)

type ClientMessage struct {
	Type   MessageType `json:"type"`
	RoomID string      `json:"roomId,omitempty"`
}

type ServerMessage struct {
	Type            MessageType `json:"type"`
	RoomID          string      `json:"roomId,omitempty"`
	UserID          string      `json:"userId,omitempty"`
	SlotID          string      `json:"slotId,omitempty"`
	BookingID       string      `json:"bookingId,omitempty"`
	ReservationID   string      `json:"reservationId,omitempty"`
	WaitlistEntryID string      `json:"waitlistEntryId,omitempty"`
	ExpiresAt       string      `json:"expiresAt,omitempty"`
	Timestamp       string      `json:"timestamp,omitempty"`
	Message         string      `json:"message,omitempty"`
}
