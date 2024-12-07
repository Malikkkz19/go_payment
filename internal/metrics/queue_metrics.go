package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// Метрики для сообщений
	MessagePublished = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "payment_queue_messages_published_total",
			Help: "The total number of messages published to queues",
		},
		[]string{"queue", "status"},
	)

	MessageConsumed = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "payment_queue_messages_consumed_total",
			Help: "The total number of messages consumed from queues",
		},
		[]string{"queue", "status"},
	)

	MessageProcessingTime = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "payment_queue_message_processing_duration_seconds",
			Help:    "Time spent processing messages",
			Buckets: prometheus.ExponentialBuckets(0.01, 2, 10),
		},
		[]string{"queue", "status"},
	)

	// Метрики для очередей
	QueueSize = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "payment_queue_size",
			Help: "The current number of messages in the queue",
		},
		[]string{"queue"},
	)

	QueueConsumers = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "payment_queue_consumers",
			Help: "The current number of consumers for the queue",
		},
		[]string{"queue"},
	)

	// Метрики для ошибок
	ProcessingErrors = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "payment_queue_processing_errors_total",
			Help: "The total number of processing errors",
		},
		[]string{"queue", "error_type"},
	)

	RetryAttempts = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "payment_queue_retry_attempts_total",
			Help: "The total number of retry attempts",
		},
		[]string{"queue"},
	)

	DeadLetterMessages = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "payment_queue_dead_letter_messages_total",
			Help: "The total number of messages sent to dead letter queue",
		},
		[]string{"queue", "reason"},
	)
)
