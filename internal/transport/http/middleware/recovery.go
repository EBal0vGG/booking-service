package middleware

import (
	"log/slog"
	"net/http"
	"runtime/debug"
	"time"

	observabilitymetrics "booking-service/internal/observability/metrics"
	"booking-service/internal/transport/http/response"

	"booking-service/internal/domain"
)

func Recovery(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		startedAt := time.Now()
		recorder := &statusRecorder{ResponseWriter: w}

		defer func() {
			recovered := recover()
			if recovered == nil {
				return
			}

			path := routePattern(r)
			if path == "" {
				path = r.URL.Path
			}

			// Write canonical error response when panic happened before any response bytes.
			if recorder.status == 0 && recorder.bytes == 0 {
				response.WriteError(recorder, domain.NewDomainError(domain.ErrorInternalError, "internal server error"))
			} else if recorder.status == 0 {
				recorder.WriteHeader(http.StatusInternalServerError)
			}

			observabilitymetrics.ObserveHTTPRequest(r.Method, path, recorder.Status(), time.Since(startedAt))

			attrs := []any{
				"method", r.Method,
				"path", path,
				"status", recorder.Status(),
				"duration_ms", time.Since(startedAt).Milliseconds(),
				"remote_addr", r.RemoteAddr,
				"request_id", requestIDFromRequest(r),
				"panic", recovered,
				"stacktrace", string(debug.Stack()),
			}
			if user, ok := UserFromContext(r.Context()); ok {
				attrs = append(attrs, "user_id", user.ID.String())
			}

			slog.Error("http_panic", attrs...)
		}()

		next.ServeHTTP(recorder, r)
	})
}
