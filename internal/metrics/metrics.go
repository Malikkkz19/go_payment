package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// PaymentRequests tracks total number of payment requests
	PaymentRequests = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "payment_requests_total",
			Help: "The total number of payment requests",
		},
		[]string{"provider", "status"},
	)

	// PaymentDuration tracks payment processing duration
	PaymentDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "payment_duration_seconds",
			Help:    "Payment processing duration in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"provider"},
	)

	// PaymentAmount tracks payment amounts
	PaymentAmount = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "payment_amount",
			Help:    "Payment amounts processed",
			Buckets: []float64{10, 50, 100, 500, 1000, 5000, 10000},
		},
		[]string{"provider", "currency"},
	)

	// ActiveConnections tracks current number of active connections
	ActiveConnections = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "active_connections",
			Help: "The current number of active connections",
		},
	)

	// DatabaseQueryDuration tracks database query duration
	DatabaseQueryDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "database_query_duration_seconds",
			Help:    "Database query duration in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"operation", "table"},
	)

	// ErrorsTotal tracks total number of errors
	ErrorsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "errors_total",
			Help: "Total number of errors by type",
		},
		[]string{"type"},
	)
)
