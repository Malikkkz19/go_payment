package messaging

import (
	"context"
	"encoding/json"
	"fmt"
	"go_payment/internal/metrics"
	"go_payment/internal/models"
	"log"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

const (
	// Названия очередей
	PaymentQueue          = "payments"
	PaymentStatusQueue    = "payment_status"
	NotificationQueue     = "notifications"
	PaymentExchange      = "payment_exchange"
	PaymentStatusExchange = "payment_status_exchange"
	DeadLetterExchange   = "dead_letter_exchange"
)

// RabbitMQ представляет клиент для работы с RabbitMQ
type RabbitMQ struct {
	conn       *amqp.Connection
	channel    *amqp.Channel
	collector  *metrics.QueueCollector
}

// NewRabbitMQ создает новый экземпляр RabbitMQ клиента
func NewRabbitMQ(url string) (*RabbitMQ, error) {
	conn, err := amqp.Dial(url)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to RabbitMQ: %w", err)
	}

	ch, err := conn.Channel()
	if err != nil {
		return nil, fmt.Errorf("failed to open channel: %w", err)
	}

	rmq := &RabbitMQ{
		conn:    conn,
		channel: ch,
	}

	if err := rmq.setupExchangesAndQueues(); err != nil {
		return nil, err
	}

	// Инициализация коллектора метрик
	queues := []string{PaymentQueue, PaymentStatusQueue, NotificationQueue}
	rmq.collector = metrics.NewQueueCollector(conn, queues)
	rmq.collector.Start(context.Background())

	return rmq, nil
}

// setupExchangesAndQueues настраивает обмены и очереди
func (r *RabbitMQ) setupExchangesAndQueues() error {
	// Настройка Dead Letter Exchange
	err := r.channel.ExchangeDeclare(
		DeadLetterExchange,
		"direct",
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return fmt.Errorf("failed to declare dead letter exchange: %w", err)
	}

	// Настройка основных обменов
	exchanges := []string{PaymentExchange, PaymentStatusExchange}
	for _, exchange := range exchanges {
		err := r.channel.ExchangeDeclare(
			exchange,
			"topic",
			true,
			false,
			false,
			false,
			nil,
		)
		if err != nil {
			return fmt.Errorf("failed to declare exchange %s: %w", exchange, err)
		}
	}

	// Настройка очередей с dead-letter
	queues := []string{PaymentQueue, PaymentStatusQueue, NotificationQueue}
	for _, queue := range queues {
		args := amqp.Table{
			"x-dead-letter-exchange": DeadLetterExchange,
			"x-message-ttl":         int32(24 * time.Hour.Milliseconds()),
		}

		_, err := r.channel.QueueDeclare(
			queue,
			true,
			false,
			false,
			false,
			args,
		)
		if err != nil {
			return fmt.Errorf("failed to declare queue %s: %w", queue, err)
		}
	}

	// Привязка очередей к обменам
	err = r.channel.QueueBind(
		PaymentQueue,
		"payment.#",
		PaymentExchange,
		false,
		nil,
	)
	if err != nil {
		return fmt.Errorf("failed to bind payment queue: %w", err)
	}

	err = r.channel.QueueBind(
		PaymentStatusQueue,
		"status.#",
		PaymentStatusExchange,
		false,
		nil,
	)
	if err != nil {
		return fmt.Errorf("failed to bind status queue: %w", err)
	}

	return nil
}

// PublishPayment публикует сообщение о платеже
func (r *RabbitMQ) PublishPayment(ctx context.Context, msg *models.PaymentMessage) error {
	body, err := json.Marshal(msg)
	if err != nil {
		metrics.RecordProcessingError(PaymentQueue, "marshal_error")
		return fmt.Errorf("failed to marshal payment message: %w", err)
	}

	err = r.channel.PublishWithContext(ctx,
		PaymentExchange,
		fmt.Sprintf("payment.%s", msg.Provider),
		false,
		false,
		amqp.Publishing{
			ContentType:  "application/json",
			Body:        body,
			MessageId:   msg.OrderID,
			Timestamp:   time.Now(),
			DeliveryMode: amqp.Persistent,
		},
	)

	if err != nil {
		metrics.RecordProcessingError(PaymentQueue, "publish_error")
		return err
	}

	metrics.IncrementPublishedMessage(PaymentQueue, string(msg.Status))
	return nil
}

// PublishPaymentStatus публикует сообщение об изменении статуса платежа
func (r *RabbitMQ) PublishPaymentStatus(ctx context.Context, msg *models.PaymentStatusMessage) error {
	body, err := json.Marshal(msg)
	if err != nil {
		metrics.RecordProcessingError(PaymentStatusQueue, "marshal_error")
		return fmt.Errorf("failed to marshal status message: %w", err)
	}

	err = r.channel.PublishWithContext(ctx,
		PaymentStatusExchange,
		fmt.Sprintf("status.%s", msg.OrderID),
		false,
		false,
		amqp.Publishing{
			ContentType:  "application/json",
			Body:        body,
			MessageId:   msg.OrderID,
			Timestamp:   time.Now(),
			DeliveryMode: amqp.Persistent,
		},
	)

	if err != nil {
		metrics.RecordProcessingError(PaymentStatusQueue, "publish_error")
		return err
	}

	metrics.IncrementPublishedMessage(PaymentStatusQueue, string(msg.NewStatus))
	return nil
}

// PublishNotification публикует уведомление
func (r *RabbitMQ) PublishNotification(ctx context.Context, msg *models.NotificationMessage) error {
	body, err := json.Marshal(msg)
	if err != nil {
		metrics.RecordProcessingError(NotificationQueue, "marshal_error")
		return fmt.Errorf("failed to marshal notification message: %w", err)
	}

	err = r.channel.PublishWithContext(ctx,
		"",
		NotificationQueue,
		false,
		false,
		amqp.Publishing{
			ContentType:  "application/json",
			Body:        body,
			MessageId:   msg.ID,
			Timestamp:   time.Now(),
			DeliveryMode: amqp.Persistent,
		},
	)

	if err != nil {
		metrics.RecordProcessingError(NotificationQueue, "publish_error")
		return err
	}

	metrics.IncrementPublishedMessage(NotificationQueue, string(msg.Status))
	return nil
}

// ConsumePayments начинает потребление сообщений о платежах
func (r *RabbitMQ) ConsumePayments(handler func(msg *models.PaymentMessage) error) error {
	msgs, err := r.channel.Consume(
		PaymentQueue,
		"",
		false,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return fmt.Errorf("failed to register a consumer: %w", err)
	}

	go func() {
		for d := range msgs {
			var paymentMsg models.PaymentMessage
			done := metrics.TrackProcessingTime(PaymentQueue, "processing")

			if err := json.Unmarshal(d.Body, &paymentMsg); err != nil {
				log.Printf("Error unmarshaling payment message: %v", err)
				metrics.RecordProcessingError(PaymentQueue, "unmarshal_error")
				d.Reject(false)
				metrics.RecordDeadLetterMessage(PaymentQueue, "unmarshal_error")
				done()
				continue
			}

			if err := handler(&paymentMsg); err != nil {
				log.Printf("Error handling payment message: %v", err)
				metrics.RecordProcessingError(PaymentQueue, "handler_error")
				d.Reject(true)
				metrics.IncrementRetryAttempt(PaymentQueue)
				done()
				continue
			}

			d.Ack(false)
			metrics.IncrementProcessedMessage(PaymentQueue, string(paymentMsg.Status))
			done()
		}
	}()

	return nil
}

// ConsumePaymentStatus начинает потребление сообщений о статусах платежей
func (r *RabbitMQ) ConsumePaymentStatus(handler func(msg *models.PaymentStatusMessage) error) error {
	msgs, err := r.channel.Consume(
		PaymentStatusQueue,
		"",
		false,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return fmt.Errorf("failed to register a consumer: %w", err)
	}

	go func() {
		for d := range msgs {
			var statusMsg models.PaymentStatusMessage
			done := metrics.TrackProcessingTime(PaymentStatusQueue, "processing")

			if err := json.Unmarshal(d.Body, &statusMsg); err != nil {
				log.Printf("Error unmarshaling status message: %v", err)
				metrics.RecordProcessingError(PaymentStatusQueue, "unmarshal_error")
				d.Reject(false)
				metrics.RecordDeadLetterMessage(PaymentStatusQueue, "unmarshal_error")
				done()
				continue
			}

			if err := handler(&statusMsg); err != nil {
				log.Printf("Error handling status message: %v", err)
				metrics.RecordProcessingError(PaymentStatusQueue, "handler_error")
				d.Reject(true)
				metrics.IncrementRetryAttempt(PaymentStatusQueue)
				done()
				continue
			}

			d.Ack(false)
			metrics.IncrementProcessedMessage(PaymentStatusQueue, string(statusMsg.NewStatus))
			done()
		}
	}()

	return nil
}

// ConsumeNotifications начинает потребление уведомлений
func (r *RabbitMQ) ConsumeNotifications(handler func(msg *models.NotificationMessage) error) error {
	msgs, err := r.channel.Consume(
		NotificationQueue,
		"",
		false,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return fmt.Errorf("failed to register a consumer: %w", err)
	}

	go func() {
		for d := range msgs {
			var notificationMsg models.NotificationMessage
			done := metrics.TrackProcessingTime(NotificationQueue, "processing")

			if err := json.Unmarshal(d.Body, &notificationMsg); err != nil {
				log.Printf("Error unmarshaling notification message: %v", err)
				metrics.RecordProcessingError(NotificationQueue, "unmarshal_error")
				d.Reject(false)
				metrics.RecordDeadLetterMessage(NotificationQueue, "unmarshal_error")
				done()
				continue
			}

			if err := handler(&notificationMsg); err != nil {
				log.Printf("Error handling notification message: %v", err)
				metrics.RecordProcessingError(NotificationQueue, "handler_error")
				d.Reject(true)
				metrics.IncrementRetryAttempt(NotificationQueue)
				done()
				continue
			}

			d.Ack(false)
			metrics.IncrementProcessedMessage(NotificationQueue, string(notificationMsg.Status))
			done()
		}
	}()

	return nil
}

// Close закрывает соединение с RabbitMQ
func (r *RabbitMQ) Close() error {
	r.collector.Stop()
	
	if err := r.channel.Close(); err != nil {
		return fmt.Errorf("failed to close channel: %w", err)
	}
	if err := r.conn.Close(); err != nil {
		return fmt.Errorf("failed to close connection: %w", err)
	}
	return nil
}
