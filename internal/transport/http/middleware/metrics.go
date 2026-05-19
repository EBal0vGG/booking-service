package middleware

import (
	"net/http"
	"time"

	observemetrics "booking-service/internal/observability/metrics"
)

func HTTPMetrics(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		startedAt := time.Now()
		recorder := &statusRecorder{ResponseWriter: w}

		next.ServeHTTP(recorder, r)

		path := routePattern(r)
		if path == "" {
			path = "unknown"
		}
		observemetrics.ObserveHTTPRequest(r.Method, path, recorder.Status(), time.Since(startedAt))
	})
}
