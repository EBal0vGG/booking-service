package realtime

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	observabilitymetrics "booking-service/internal/observability/metrics"

	"github.com/redis/go-redis/v9"
)

const redisSubscriberRetryDelay = 2 * time.Second

type RedisSubscriber struct {
	client     redis.UniversalClient
	channel    string
	hub        *Hub
	retryDelay time.Duration
}

func NewRedisSubscriber(client redis.UniversalClient, channel string, hub *Hub) *RedisSubscriber {
	return &RedisSubscriber{
		client:     client,
		channel:    channel,
		hub:        hub,
		retryDelay: redisSubscriberRetryDelay,
	}
}

func (s *RedisSubscriber) Run(ctx context.Context) error {
	if s == nil {
		return nil
	}
	if s.client == nil {
		return errors.New("redis subscriber client is nil")
	}
	if s.hub == nil {
		return errors.New("redis subscriber hub is nil")
	}
	if s.channel == "" {
		return errors.New("redis subscriber channel is empty")
	}

	for {
		if ctx.Err() != nil {
			return nil
		}

		err := s.consume(ctx)
		if err == nil || ctx.Err() != nil {
			return nil
		}

		observabilitymetrics.IncRedisRealtimeSubscriberReconnect()
		slog.Warn("realtime redis subscriber consume error; retrying", "retry_delay", s.retryDelay.String(), "error", err)
		timer := time.NewTimer(s.retryDelay)
		select {
		case <-ctx.Done():
			if !timer.Stop() {
				<-timer.C
			}
			return nil
		case <-timer.C:
		}
	}
}

func (s *RedisSubscriber) consume(ctx context.Context) error {
	pubsub := s.client.Subscribe(ctx, s.channel)
	defer func() {
		if err := pubsub.Close(); err != nil && !errors.Is(err, context.Canceled) {
			slog.Warn("realtime redis subscriber close error", "error", err)
		}
	}()

	if _, err := pubsub.Receive(ctx); err != nil {
		return fmt.Errorf("subscribe channel %s: %w", s.channel, err)
	}

	ch := pubsub.Channel()
	for {
		select {
		case <-ctx.Done():
			return nil
		case msg, ok := <-ch:
			if !ok {
				return errors.New("redis pubsub channel closed")
			}
			if err := s.handleMessage(msg.Payload); err != nil {
				slog.Warn("realtime redis subscriber message dropped", "error", err)
			}
		}
	}
}

func (s *RedisSubscriber) handleMessage(payload string) error {
	var event Event
	if err := json.Unmarshal([]byte(payload), &event); err != nil {
		observabilitymetrics.IncRedisRealtimeEventReceived("unknown", "error")
		return fmt.Errorf("unmarshal event: %w", err)
	}
	if err := event.Validate(); err != nil {
		observabilitymetrics.IncRedisRealtimeEventReceived(eventTypeLabel(event.Type), "error")
		return fmt.Errorf("invalid event: %w", err)
	}

	serverMessage := event.ToServerMessage()
	outboundPayload, err := json.Marshal(serverMessage)
	if err != nil {
		observabilitymetrics.IncRedisRealtimeEventReceived(eventTypeLabel(event.Type), "error")
		return fmt.Errorf("marshal server message: %w", err)
	}

	switch event.Target {
	case EventTargetRoom:
		roomID, err := event.RoomUUID()
		if err != nil {
			observabilitymetrics.IncRedisRealtimeEventReceived(eventTypeLabel(event.Type), "error")
			return fmt.Errorf("invalid room uuid: %w", err)
		}
		s.hub.Broadcast(roomID, outboundPayload)
	case EventTargetUser:
		userID, err := event.UserUUID()
		if err != nil {
			observabilitymetrics.IncRedisRealtimeEventReceived(eventTypeLabel(event.Type), "error")
			return fmt.Errorf("invalid user uuid: %w", err)
		}
		s.hub.SendToUser(userID, outboundPayload)
	default:
		observabilitymetrics.IncRedisRealtimeEventReceived(eventTypeLabel(event.Type), "error")
		return fmt.Errorf("unsupported target: %s", event.Target)
	}
	observabilitymetrics.IncRedisRealtimeEventReceived(eventTypeLabel(event.Type), "success")
	return nil
}

func eventTypeLabel(eventType EventType) string {
	if eventType == "" {
		return "unknown"
	}
	return string(eventType)
}
