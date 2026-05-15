package realtime

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

type RedisPublisher struct {
	client  redis.UniversalClient
	channel string
}

const redisPublishTimeout = 500 * time.Millisecond

func NewRedisPublisher(client redis.UniversalClient, channel string) *RedisPublisher {
	return &RedisPublisher{
		client:  client,
		channel: channel,
	}
}

func (p *RedisPublisher) SlotBooked(ctx context.Context, roomID, slotID, bookingID uuid.UUID) {
	p.publish(ctx, EventTypeSlotBooked, roomID, slotID, bookingID)
}

func (p *RedisPublisher) SlotReleased(ctx context.Context, roomID, slotID, bookingID uuid.UUID) {
	p.publish(ctx, EventTypeSlotReleased, roomID, slotID, bookingID)
}

func (p *RedisPublisher) publish(ctx context.Context, eventType EventType, roomID, slotID, bookingID uuid.UUID) {
	_ = ctx // publish is intentionally decoupled from request lifecycle (best-effort realtime).

	if p == nil || p.client == nil {
		return
	}
	if p.channel == "" {
		log.Printf("realtime redis publish skipped: empty channel")
		return
	}

	event := NewEvent(eventType, roomID, slotID, bookingID, time.Now().UTC())
	payload, err := json.Marshal(event)
	if err != nil {
		log.Printf("realtime redis publish marshal error: %v", err)
		return
	}

	publishCtx, cancel := context.WithTimeout(context.Background(), redisPublishTimeout)
	defer cancel()

	if err := p.client.Publish(publishCtx, p.channel, payload).Err(); err != nil {
		// Best-effort delivery: booking/cancel must not fail because of realtime transport issues.
		log.Printf("realtime redis publish error channel=%s type=%s roomId=%s: %v", p.channel, event.Type, event.RoomID, err)
	}
}
