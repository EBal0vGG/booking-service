package realtime

import (
	"encoding/json"
	"sync"

	"github.com/google/uuid"
)

type Hub struct {
	mu    sync.RWMutex
	rooms map[uuid.UUID]map[*Client]struct{}
	users map[uuid.UUID]map[*Client]struct{}
}

func NewHub() *Hub {
	return &Hub{
		rooms: make(map[uuid.UUID]map[*Client]struct{}),
		users: make(map[uuid.UUID]map[*Client]struct{}),
	}
}

func (h *Hub) Subscribe(roomID uuid.UUID, c *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()

	clients, ok := h.rooms[roomID]
	if !ok {
		clients = make(map[*Client]struct{})
		h.rooms[roomID] = clients
	}
	clients[c] = struct{}{}
}

func (h *Hub) Unsubscribe(roomID uuid.UUID, c *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()

	clients, ok := h.rooms[roomID]
	if !ok {
		return
	}
	delete(clients, c)
	if len(clients) == 0 {
		delete(h.rooms, roomID)
	}
}

func (h *Hub) UnsubscribeAll(c *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()

	for roomID, clients := range h.rooms {
		delete(clients, c)
		if len(clients) == 0 {
			delete(h.rooms, roomID)
		}
	}
}

func (h *Hub) RegisterUser(userID uuid.UUID, c *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()

	clients, ok := h.users[userID]
	if !ok {
		clients = make(map[*Client]struct{})
		h.users[userID] = clients
	}
	clients[c] = struct{}{}
}

func (h *Hub) UnregisterUser(userID uuid.UUID, c *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()

	clients, ok := h.users[userID]
	if !ok {
		return
	}
	delete(clients, c)
	if len(clients) == 0 {
		delete(h.users, userID)
	}
}

func (h *Hub) Broadcast(roomID uuid.UUID, payload []byte) {
	h.mu.RLock()
	clientsMap := h.rooms[roomID]
	clients := make([]*Client, 0, len(clientsMap))
	for c := range clientsMap {
		clients = append(clients, c)
	}
	h.mu.RUnlock()

	messageType := extractMessageType(payload)
	slow := make([]*Client, 0)
	for _, c := range clients {
		if !c.enqueue(payload, messageType) {
			slow = append(slow, c)
		}
	}

	// Disconnect slow consumers to avoid global backpressure.
	for _, c := range slow {
		c.Close()
	}
}

func (h *Hub) SendToUser(userID uuid.UUID, payload []byte) {
	h.mu.RLock()
	clientsMap := h.users[userID]
	clients := make([]*Client, 0, len(clientsMap))
	for c := range clientsMap {
		clients = append(clients, c)
	}
	h.mu.RUnlock()

	messageType := extractMessageType(payload)
	slow := make([]*Client, 0)
	for _, c := range clients {
		if !c.enqueue(payload, messageType) {
			slow = append(slow, c)
		}
	}
	for _, c := range slow {
		c.Close()
	}
}

func extractMessageType(payload []byte) string {
	var message struct {
		Type string `json:"type"`
	}
	if err := json.Unmarshal(payload, &message); err != nil {
		return "unknown"
	}
	if message.Type == "" {
		return "unknown"
	}
	return message.Type
}
