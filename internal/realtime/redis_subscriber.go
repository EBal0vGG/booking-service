package realtime

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"time"

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

		log.Printf("realtime redis subscriber consume error, retrying in %s: %v", s.retryDelay, err)
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
			log.Printf("realtime redis subscriber close error: %v", err)
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
				log.Printf("realtime redis subscriber message dropped: %v", err)
			}
		}
	}
}

func (s *RedisSubscriber) handleMessage(payload string) error {
	var event Event
	if err := json.Unmarshal([]byte(payload), &event); err != nil {
		return fmt.Errorf("unmarshal event: %w", err)
	}
	if err := event.Validate(); err != nil {
		return fmt.Errorf("invalid event: %w", err)
	}

	roomID, err := event.RoomUUID()
	if err != nil {
		return fmt.Errorf("invalid room uuid: %w", err)
	}

	serverMessage := event.ToServerMessage()
	outboundPayload, err := json.Marshal(serverMessage)
	if err != nil {
		return fmt.Errorf("marshal server message: %w", err)
	}

	s.hub.Broadcast(roomID, outboundPayload)
	return nil
}
