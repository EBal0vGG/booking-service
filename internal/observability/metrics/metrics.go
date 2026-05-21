package metrics

import (
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	httpRequestsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "http_requests_total",
		Help: "Total HTTP requests processed.",
	}, []string{"method", "path", "status"})

	httpRequestDurationSeconds = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "http_request_duration_seconds",
		Help:    "HTTP request latency in seconds.",
		Buckets: prometheus.DefBuckets,
	}, []string{"method", "path", "status"})

	bookingCreatedTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "booking_created_total",
		Help: "Total successful booking creations.",
	})
	bookingCancelledTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "booking_cancelled_total",
		Help: "Total successful booking cancellations.",
	})
	bookingConflictsTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "booking_conflicts_total",
		Help: "Total booking create conflicts.",
	})
	bookingCreateErrorsTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "booking_create_errors_total",
		Help: "Total booking create errors excluding conflicts.",
	})
	bookingCancelErrorsTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "booking_cancel_errors_total",
		Help: "Total booking cancel errors.",
	})

	wsConnectionsCurrent = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "ws_connections_current",
		Help: "Current number of websocket connections.",
	})
	wsMessagesSentTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "ws_messages_sent_total",
		Help: "Total websocket messages sent to clients by message type.",
	}, []string{"type"})
	wsSubscriptionsCurrent = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "ws_subscriptions_current",
		Help: "Current number of active websocket room subscriptions.",
	})

	redisRealtimePublishTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "redis_realtime_publish_total",
		Help: "Total realtime events publish attempts to Redis by type and result.",
	}, []string{"type", "result"})
	redisRealtimeEventsReceivedTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "redis_realtime_events_received_total",
		Help: "Total realtime events received from Redis by type and processing result.",
	}, []string{"type", "result"})
	redisRealtimeSubscriberReconnectsTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "redis_realtime_subscriber_reconnects_total",
		Help: "Total Redis realtime subscriber reconnect attempts.",
	})

	waitlistJoinedTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "waitlist_joined_total",
		Help: "Total successful waitlist joins.",
	})
	waitlistNotificationsTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "waitlist_notifications_total",
		Help: "Total waitlist notifications sent (active -> notified).",
	})
	waitlistCancelledTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "waitlist_cancelled_total",
		Help: "Total waitlist leaves causing cancelled status.",
	})
)

func ObserveHTTPRequest(method, path string, status int, duration time.Duration) {
	statusLabel := strconv.Itoa(status)
	httpRequestsTotal.WithLabelValues(method, path, statusLabel).Inc()
	httpRequestDurationSeconds.WithLabelValues(method, path, statusLabel).Observe(duration.Seconds())
}

type BookingUsecaseMetrics struct{}

func NewBookingUsecaseMetrics() *BookingUsecaseMetrics {
	return &BookingUsecaseMetrics{}
}

func (BookingUsecaseMetrics) IncBookingCreated() {
	bookingCreatedTotal.Inc()
}

func (BookingUsecaseMetrics) IncBookingCancelled() {
	bookingCancelledTotal.Inc()
}

func (BookingUsecaseMetrics) IncBookingConflict() {
	bookingConflictsTotal.Inc()
}

func (BookingUsecaseMetrics) IncBookingCreateError() {
	bookingCreateErrorsTotal.Inc()
}

func (BookingUsecaseMetrics) IncBookingCancelError() {
	bookingCancelErrorsTotal.Inc()
}

func IncWSConnections() {
	wsConnectionsCurrent.Inc()
}

func DecWSConnections() {
	wsConnectionsCurrent.Dec()
}

func AddWSSubscriptions(delta int) {
	wsSubscriptionsCurrent.Add(float64(delta))
}

func IncWSMessageSent(messageType string) {
	wsMessagesSentTotal.WithLabelValues(sanitizeLabel(messageType, "unknown")).Inc()
}

func IncRedisRealtimePublish(eventType, result string) {
	redisRealtimePublishTotal.WithLabelValues(
		sanitizeLabel(eventType, "unknown"),
		sanitizeLabel(result, "unknown"),
	).Inc()
}

func IncRedisRealtimeEventReceived(eventType, result string) {
	redisRealtimeEventsReceivedTotal.WithLabelValues(
		sanitizeLabel(eventType, "unknown"),
		sanitizeLabel(result, "unknown"),
	).Inc()
}

func IncRedisRealtimeSubscriberReconnect() {
	redisRealtimeSubscriberReconnectsTotal.Inc()
}

func IncWaitlistJoined() {
	waitlistJoinedTotal.Inc()
}

func IncWaitlistNotification() {
	waitlistNotificationsTotal.Inc()
}

func IncWaitlistCancelled() {
	waitlistCancelledTotal.Inc()
}

func sanitizeLabel(value, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}
