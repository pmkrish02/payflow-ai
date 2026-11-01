package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	HTTPRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{Name: "http_requests_total", Help: "Total HTTP requests."},
		[]string{"method", "path", "status"},
	)

	HTTPRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{Name: "http_request_duration_seconds", Help: "HTTP request latency.", Buckets: prometheus.DefBuckets},
		[]string{"method", "path"},
	)

	TransfersTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{Name: "transfers_total", Help: "Transfers by result."},
		[]string{"result"},
	)

	LLMRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{Name: "llm_request_duration_seconds", Help: "Gemini call latency.", Buckets: []float64{0.25, 0.5, 1, 2, 4, 8}},
		[]string{"agent", "step"},
	)

	LLMTokensTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{Name: "llm_tokens_total", Help: "Gemini tokens used."},
		[]string{"agent", "type"},
	)
)