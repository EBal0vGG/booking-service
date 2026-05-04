package realtime

import (
	"encoding/json"
	"sync"
	"time"

	"booking-service/internal/domain"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

const (
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	pingPeriod     = (pongWait * 9) / 10
	maxMessageSize = 1024

	outboundQueueSize = 64
	maxSubscriptions  = 100
)

type Client struct {
	conn *websocket.Conn
	hub  *Hub
	user domain.User

	send chan []byte

	mu         sync.Mutex
	subscribed map[uuid.UUID]struct{}
	closeOnce  sync.Once
}

func NewClient(conn *websocket.Conn, hub *Hub, user domain.User) *Client {
	return &Client{
		conn:       conn,
		hub:        hub,
		user:       user,
		send:       make(chan []byte, outboundQueueSize),
		subscribed: make(map[uuid.UUID]struct{}),
	}
}

func (c *Client) Run() {
	go c.writePump()
	c.readPump()
}

func (c *Client) Close() {
	c.closeOnce.Do(func() {
		_ = c.conn.Close()
		c.hub.UnsubscribeAll(c)
		close(c.send)
	})
}

func (c *Client) readPump() {
	defer c.Close()

	c.conn.SetReadLimit(maxMessageSize)
	_ = c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		return c.conn.SetReadDeadline(time.Now().Add(pongWait))
	})

	for {
		_, data, err := c.conn.ReadMessage()
		if err != nil {
			// Normal close path or read error.
			break
		}

		var msg ClientMessage
		if err := json.Unmarshal(data, &msg); err != nil {
			c.sendError("invalid message")
			continue
		}

		switch msg.Type {
		case MessageTypeSubscribe:
			roomID, err := uuid.Parse(msg.RoomID)
			if err != nil {
				c.sendError("invalid roomId")
				continue
			}
			c.subscribe(roomID)
		case MessageTypeUnsubscribe:
			roomID, err := uuid.Parse(msg.RoomID)
			if err != nil {
				c.sendError("invalid roomId")
				continue
			}
			c.unsubscribe(roomID)
		default:
			c.sendError("unsupported message type")
		}
	}
}

func (c *Client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.Close()
	}()

	for {
		select {
		case msg, ok := <-c.send:
			_ = c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				_ = c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			if err := c.conn.WriteMessage(websocket.TextMessage, msg); err != nil {
				return
			}
		case <-ticker.C:
			_ = c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

func (c *Client) subscribe(roomID uuid.UUID) {
	c.mu.Lock()
	if _, ok := c.subscribed[roomID]; ok {
		c.mu.Unlock()
		return
	}
	if len(c.subscribed) >= maxSubscriptions {
		c.mu.Unlock()
		c.sendError("too many subscriptions")
		return
	}
	c.subscribed[roomID] = struct{}{}
	c.mu.Unlock()

	c.hub.Subscribe(roomID, c)
	c.sendServerMessage(ServerMessage{
		Type:   MessageTypeSubscribed,
		RoomID: roomID.String(),
	})
}

func (c *Client) unsubscribe(roomID uuid.UUID) {
	c.mu.Lock()
	if _, ok := c.subscribed[roomID]; !ok {
		c.mu.Unlock()
		return
	}
	delete(c.subscribed, roomID)
	c.mu.Unlock()

	c.hub.Unsubscribe(roomID, c)
	c.sendServerMessage(ServerMessage{
		Type:   MessageTypeUnsubscribed,
		RoomID: roomID.String(),
	})
}

func (c *Client) sendError(msg string) {
	c.sendServerMessage(ServerMessage{
		Type:    MessageTypeError,
		Message: msg,
	})
}

func (c *Client) sendServerMessage(msg ServerMessage) {
	payload, err := json.Marshal(msg)
	if err != nil {
		return
	}
	if !c.enqueue(payload) {
		// Drop slow client.
		c.Close()
	}
}

func (c *Client) enqueue(payload []byte) (ok bool) {
	defer func() {
		if recover() != nil {
			ok = false
		}
	}()
	select {
	case c.send <- payload:
		return true
	default:
		return false
	}
}
