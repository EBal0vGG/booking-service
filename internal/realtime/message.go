package realtime

type MessageType string

const (
	MessageTypeSubscribe   MessageType = "subscribe"
	MessageTypeUnsubscribe MessageType = "unsubscribe"

	MessageTypeSubscribed            MessageType = "subscribed"
	MessageTypeUnsubscribed          MessageType = "unsubscribed"
	MessageTypeSlotBooked            MessageType = "slot_booked"
	MessageTypeSlotReleased          MessageType = "slot_released"
	MessageTypeWaitlistSlotAvailable MessageType = "waitlist_slot_available"
	MessageTypeError                 MessageType = "error"
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
	WaitlistEntryID string      `json:"waitlistEntryId,omitempty"`
	Timestamp       string      `json:"timestamp,omitempty"`
	Message         string      `json:"message,omitempty"`
}
