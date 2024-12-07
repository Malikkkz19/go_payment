package metrics

import (
	"context"
	"log"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

// QueueCollector собирает метрики очередей
type QueueCollector struct {
	conn   *amqp.Connection
	queues []string
	done   chan struct{}
}

// NewQueueCollector создает новый коллектор метрик
func NewQueueCollector(conn *amqp.Connection, queues []string) *QueueCollector {
	return &QueueCollector{
		conn:   conn,
		queues: queues,
		done:   make(chan struct{}),
	}
}

// Start запускает сбор метрик
func (c *QueueCollector) Start(ctx context.Context) {
	go func() {
		ticker := time.NewTicker(15 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if err := c.collectMetrics(); err != nil {
					log.Printf("Error collecting queue metrics: %v", err)
				}
			}
		}
	}()
}

// Stop останавливает сбор метрик
func (c *QueueCollector) Stop() {
	close(c.done)
}

// collectMetrics собирает метрики очередей
func (c *QueueCollector) collectMetrics() error {
	ch, err := c.conn.Channel()
	if err != nil {
		return err
	}
	defer ch.Close()

	for _, queueName := range c.queues {
		queue, err := ch.QueueInspect(queueName)
		if err != nil {
			log.Printf("Error inspecting queue %s: %v", queueName, err)
			continue
		}

		// Обновляем метрики
		QueueSize.WithLabelValues(queueName).Set(float64(queue.Messages))
		QueueConsumers.WithLabelValues(queueName).Set(float64(queue.Consumers))

		// Если очередь пуста и есть сообщения в unacked, возможно есть проблемы с обработкой
		if queue.Messages == 0 && queue.Consumers > 0 && queue.MessagesUnacknowledged > 0 {
			log.Printf("Warning: Queue %s has %d unacknowledged messages", 
				queueName, queue.MessagesUnacknowledged)
		}
	}

	return nil
}

// TrackProcessingTime измеряет время обработки сообщения
func TrackProcessingTime(queue, status string) func() {
	start := time.Now()
	return func() {
		duration := time.Since(start).Seconds()
		MessageProcessingTime.WithLabelValues(queue, status).Observe(duration)
	}
}

// IncrementProcessedMessage увеличивает счетчик обработанных сообщений
func IncrementProcessedMessage(queue, status string) {
	MessageConsumed.WithLabelValues(queue, status).Inc()
}

// IncrementPublishedMessage увеличивает счетчик опубликованных сообщений
func IncrementPublishedMessage(queue, status string) {
	MessagePublished.WithLabelValues(queue, status).Inc()
}

// RecordProcessingError записывает ошибку обработки
func RecordProcessingError(queue, errorType string) {
	ProcessingErrors.WithLabelValues(queue, errorType).Inc()
}

// IncrementRetryAttempt увеличивает счетчик попыток повтора
func IncrementRetryAttempt(queue string) {
	RetryAttempts.WithLabelValues(queue).Inc()
}

// RecordDeadLetterMessage записывает сообщение в dead letter queue
func RecordDeadLetterMessage(queue, reason string) {
	DeadLetterMessages.WithLabelValues(queue, reason).Inc()
}
