package service

import (
	"context"
	"fmt"
	"go_payment/internal/errors"
	"go_payment/internal/messaging"
	"go_payment/internal/models"
	"go_payment/internal/retry"
	"log"
	"time"

	"gorm.io/gorm"
)

// AsyncService обрабатывает асинхронные операции
type AsyncService struct {
	db       *gorm.DB
	rabbitmq *messaging.RabbitMQ
	strategy *errors.RetryStrategy
}

// NewAsyncService создает новый экземпляр AsyncService
func NewAsyncService(db *gorm.DB, rabbitmq *messaging.RabbitMQ) *AsyncService {
	return &AsyncService{
		db:       db,
		rabbitmq: rabbitmq,
		strategy: errors.DefaultRetryStrategy(),
	}
}

// StartProcessing запускает обработку асинхронных операций
func (s *AsyncService) StartProcessing() error {
	// Обработка платежей
	if err := s.rabbitmq.ConsumePayments(s.handlePayment); err != nil {
		return fmt.Errorf("failed to start payment consumer: %w", err)
	}

	// Обработка изменений статуса
	if err := s.rabbitmq.ConsumePaymentStatus(s.handlePaymentStatus); err != nil {
		return fmt.Errorf("failed to start status consumer: %w", err)
	}

	log.Println("Started processing async operations")
	return nil
}

// handlePayment обрабатывает сообщение о платеже
func (s *AsyncService) handlePayment(msg *models.PaymentMessage) error {
	ctx := context.Background()

	// Создаем операцию для обработки платежа
	operation := func(ctx context.Context) error {
		// Проверяем, не был ли платеж уже обработан
		var existingPayment models.Payment
		if err := s.db.Where("order_id = ? AND status != ?", 
			msg.OrderID, models.PaymentStatusPending).First(&existingPayment).Error; err == nil {
			// Платеж уже обработан
			return nil
		}

		// Создаем новый платеж в базе данных
		payment := &models.Payment{
			OrderID:       msg.OrderID,
			Amount:        msg.Amount,
			Currency:      msg.Currency,
			Status:        msg.Status,
			Provider:      msg.Provider,
			CustomerID:    msg.CustomerID,
			CustomerEmail: msg.CustomerEmail,
			PaymentDate:   msg.CreatedAt,
			MetaData:      msg.MetaData,
		}

		if err := s.db.Create(payment).Error; err != nil {
			return errors.NewPaymentError(
				errors.ErrorTypeDatabase,
				"DB_ERROR",
				"Failed to create payment record",
				msg.OrderID,
				true,
				err,
			)
		}

		// Отправляем уведомление о создании платежа
		notification := &models.NotificationMessage{
			Type:      "email",
			Recipient: msg.CustomerEmail,
			Subject:   fmt.Sprintf("Payment Received - Order %s", msg.OrderID),
			Content:   fmt.Sprintf("We have received your payment of %.2f %s", msg.Amount, msg.Currency),
			CreatedAt: time.Now(),
		}

		if err := s.rabbitmq.PublishNotification(ctx, notification); err != nil {
			// Ошибка отправки уведомления не должна влиять на обработку платежа
			log.Printf("Failed to send notification for order %s: %v", msg.OrderID, err)
		}

		return nil
	}

	// Выполняем операцию с повторными попытками
	return retry.WithRetry(ctx, operation, s.strategy)
}

// handlePaymentStatus обрабатывает изменение статуса платежа
func (s *AsyncService) handlePaymentStatus(msg *models.PaymentStatusMessage) error {
	ctx := context.Background()

	// Создаем операцию для обновления статуса
	operation := func(ctx context.Context) error {
		// Проверяем существование платежа
		var payment models.Payment
		if err := s.db.Where("order_id = ?", msg.OrderID).First(&payment).Error; err != nil {
			return errors.NewPaymentError(
				errors.ErrorTypeDatabase,
				"PAYMENT_NOT_FOUND",
				"Payment not found",
				msg.OrderID,
				false,
				err,
			)
		}

		// Обновляем статус платежа
		if err := s.db.Model(&payment).Update("status", msg.NewStatus).Error; err != nil {
			return errors.NewPaymentError(
				errors.ErrorTypeDatabase,
				"STATUS_UPDATE_ERROR",
				"Failed to update payment status",
				msg.OrderID,
				true,
				err,
			)
		}

		// Отправляем уведомление об изменении статуса
		notification := &models.NotificationMessage{
			Type:      "email",
			Recipient: payment.CustomerEmail,
			Subject:   fmt.Sprintf("Payment Status Updated - Order %s", msg.OrderID),
			Content:   fmt.Sprintf("Your payment status has been updated to %s", msg.NewStatus),
			CreatedAt: time.Now(),
		}

		if err := s.rabbitmq.PublishNotification(ctx, notification); err != nil {
			log.Printf("Failed to send status notification for order %s: %v", msg.OrderID, err)
		}

		return nil
	}

	// Выполняем операцию с повторными попытками
	return retry.WithRetry(ctx, operation, s.strategy)
}

// ProcessPaymentAsync асинхронно обрабатывает платеж
func (s *AsyncService) ProcessPaymentAsync(ctx context.Context, payment *models.Payment) error {
	msg := &models.PaymentMessage{
		OrderID:       payment.OrderID,
		Amount:        payment.Amount,
		Currency:      payment.Currency,
		Status:        payment.Status,
		Provider:      payment.Provider,
		CustomerID:    payment.CustomerID,
		CustomerEmail: payment.CustomerEmail,
		CreatedAt:     time.Now(),
		MetaData:      payment.MetaData,
	}

	operation := func(ctx context.Context) error {
		if err := s.rabbitmq.PublishPayment(ctx, msg); err != nil {
			return errors.NewPaymentError(
				errors.ErrorTypeMessaging,
				"PUBLISH_ERROR",
				"Failed to publish payment message",
				payment.OrderID,
				true,
				err,
			)
		}
		return nil
	}

	return retry.WithRetry(ctx, operation, s.strategy)
}

// UpdatePaymentStatusAsync асинхронно обновляет статус платежа
func (s *AsyncService) UpdatePaymentStatusAsync(ctx context.Context, orderID string, oldStatus, newStatus models.PaymentStatus) error {
	msg := &models.PaymentStatusMessage{
		OrderID:   orderID,
		OldStatus: oldStatus,
		NewStatus: newStatus,
		UpdatedAt: time.Now(),
	}

	operation := func(ctx context.Context) error {
		if err := s.rabbitmq.PublishPaymentStatus(ctx, msg); err != nil {
			return errors.NewPaymentError(
				errors.ErrorTypeMessaging,
				"STATUS_PUBLISH_ERROR",
				"Failed to publish status update message",
				orderID,
				true,
				err,
			)
		}
		return nil
	}

	return retry.WithRetry(ctx, operation, s.strategy)
}
