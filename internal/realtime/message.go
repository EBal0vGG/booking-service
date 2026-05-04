package realtime

type MessageType string

const (
	MessageTypeSubscribe   MessageType = "subscribe"
	MessageTypeUnsubscribe MessageType = "unsubscribe"

	MessageTypeSubscribed   MessageType = "subscribed"
	MessageTypeUnsubscribed MessageType = "unsubscribed"
	MessageTypeSlotBooked   MessageType = "slot_booked"
	MessageTypeSlotReleased MessageType = "slot_released"
	MessageTypeError        MessageType = "error"
)

type ClientMessage struct {
	Type   MessageType `json:"type"`
	RoomID string      `json:"roomId,omitempty"`
}

type ServerMessage struct {
	Type      MessageType `json:"type"`
	RoomID    string      `json:"roomId,omitempty"`
	SlotID    string      `json:"slotId,omitempty"`
	BookingID string      `json:"bookingId,omitempty"`
	Timestamp string      `json:"timestamp,omitempty"`
	Message   string      `json:"message,omitempty"`
}
