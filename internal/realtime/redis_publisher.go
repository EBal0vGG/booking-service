package realtime

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	observabilitymetrics "booking-service/internal/observability/metrics"

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
	p.publishEvent(ctx, NewRoomEvent(EventTypeSlotBooked, roomID, slotID, bookingID, time.Now().UTC()))
}

func (p *RedisPublisher) SlotReleased(ctx context.Context, roomID, slotID, bookingID uuid.UUID) {
	p.publishEvent(ctx, NewRoomEvent(EventTypeSlotReleased, roomID, slotID, bookingID, time.Now().UTC()))
}

func (p *RedisPublisher) WaitlistSlotAvailable(ctx context.Context, roomID, slotID, userID, waitlistEntryID uuid.UUID) {
	p.publishEvent(ctx, NewUserEvent(EventTypeWaitlistSlotAvailable, roomID, slotID, userID, waitlistEntryID, time.Now().UTC()))
}

func (p *RedisPublisher) publishEvent(ctx context.Context, event Event) {
	_ = ctx // publish is intentionally decoupled from request lifecycle (best-effort realtime).

	if p == nil || p.client == nil {
		return
	}
	if p.channel == "" {
		observabilitymetrics.IncRedisRealtimePublish(string(event.Type), "error")
		slog.Warn("realtime redis publish skipped: empty channel", "event_type", event.Type, "target", event.Target)
		return
	}

	payload, err := json.Marshal(event)
	if err != nil {
		observabilitymetrics.IncRedisRealtimePublish(string(event.Type), "error")
		slog.Error("realtime redis publish marshal error", "event_type", event.Type, "error", err)
		return
	}

	publishCtx, cancel := context.WithTimeout(context.Background(), redisPublishTimeout)
	defer cancel()

	if err := p.client.Publish(publishCtx, p.channel, payload).Err(); err != nil {
		// Best-effort delivery: booking/cancel must not fail because of realtime transport issues.
		observabilitymetrics.IncRedisRealtimePublish(string(event.Type), "error")
		slog.Warn(
			"realtime redis publish error",
			"channel", p.channel,
			"event_type", event.Type,
			"target", event.Target,
			"room_id", event.RoomID,
			"user_id", event.UserID,
			"error", err,
		)
		return
	}
	observabilitymetrics.IncRedisRealtimePublish(string(event.Type), "success")
}
