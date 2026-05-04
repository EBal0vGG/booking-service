package realtime

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type Manager struct {
	hub *Hub
}

func NewManager(hub *Hub) *Manager {
	return &Manager{hub: hub}
}

func (m *Manager) SlotBooked(ctx context.Context, roomID, slotID, bookingID uuid.UUID) {
	_ = ctx
	m.publish(MessageTypeSlotBooked, roomID, slotID, bookingID)
}

func (m *Manager) SlotReleased(ctx context.Context, roomID, slotID, bookingID uuid.UUID) {
	_ = ctx
	m.publish(MessageTypeSlotReleased, roomID, slotID, bookingID)
}

func (m *Manager) publish(msgType MessageType, roomID, slotID, bookingID uuid.UUID) {
	if m == nil || m.hub == nil {
		return
	}
	msg := ServerMessage{
		Type:      msgType,
		RoomID:    roomID.String(),
		SlotID:    slotID.String(),
		BookingID: bookingID.String(),
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}
	payload, err := json.Marshal(msg)
	if err != nil {
		return
	}
	m.hub.Broadcast(roomID, payload)
}
