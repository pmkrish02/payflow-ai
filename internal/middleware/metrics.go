package middleware

import (
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/pmkrish02/payflow-ai/internal/metrics"
)

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (s *statusRecorder) WriteHeader(code int) {
	s.status = code
	s.ResponseWriter.WriteHeader(code)
}

func Metrics(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}

		next.ServeHTTP(rec, r)

		path := "unmatched"
		if rc := chi.RouteContext(r.Context()); rc != nil && rc.RoutePattern() != "" {
			path = rc.RoutePattern()
		}

		metrics.HTTPRequestDuration.WithLabelValues(r.Method, path).Observe(time.Since(start).Seconds())
		metrics.HTTPRequestsTotal.WithLabelValues(r.Method, path, strconv.Itoa(rec.status)).Inc()
	})
}