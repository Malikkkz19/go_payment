package service

import (
	"context"
	"fmt"
	"go_payment/internal/models"
	"go_payment/internal/payment"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// PaymentService представляет сервис для работы с платежами
type PaymentService struct {
	db              *gorm.DB
	asyncService    *AsyncService
	providerFactory *payment.ProviderFactory
	providers       map[payment.ProviderType]payment.Provider
}

// NewPaymentService создает новый экземпляр сервиса платежей
func NewPaymentService(db *gorm.DB, asyncService *AsyncService) *PaymentService {
	factory := payment.NewProviderFactory()
	
	// Регистрируем поддерживаемые провайдеры
	factory.RegisterProvider(payment.ProviderStripe, payment.NewStripeProvider())
	factory.RegisterProvider(payment.ProviderPayPal, payment.NewPayPalProvider())

	return &PaymentService{
		db:              db,
		asyncService:    asyncService,
		providerFactory: factory,
		providers:       make(map[payment.ProviderType]payment.Provider),
	}
}

// InitializeProviders инициализирует платежных провайдеров
func (s *PaymentService) InitializeProviders(config map[payment.ProviderType]map[string]string) error {
	for providerType, providerConfig := range config {
		provider, err := s.providerFactory.CreateProvider(providerType, providerConfig)
		if err != nil {
			return fmt.Errorf("failed to initialize provider %s: %w", providerType, err)
		}
		s.providers[providerType] = provider
	}
	return nil
}

// ProcessPayment обрабатывает платеж
func (s *PaymentService) ProcessPayment(ctx context.Context, payment *models.Payment) error {
	// Получаем провайдера для платежа
	provider, exists := s.providers[payment.ProviderType]
	if !exists {
		return fmt.Errorf("unsupported payment provider: %s", payment.ProviderType)
	}

	// Создаем запрос к провайдеру
	req := payment.PaymentRequest{
		OrderID:       payment.OrderID,
		Amount:        payment.Amount,
		Currency:      payment.Currency,
		CustomerID:    payment.CustomerID,
		CustomerEmail: payment.CustomerEmail,
		Description:   payment.Description,
		MetaData:      payment.Metadata,
	}

	// Обрабатываем платеж через провайдера
	resp, err := provider.ProcessPayment(ctx, req)
	if err != nil {
		payment.Status = models.PaymentStatusFailed
		payment.ErrorMessage = err.Error()
		s.db.Save(payment)
		return fmt.Errorf("failed to process payment: %w", err)
	}

	// Обновляем информацию о платеже
	payment.TransactionID = resp.TransactionID
	payment.Status = resp.Status
	payment.PaymentDetails = resp.PaymentDetails
	payment.UpdatedAt = time.Now()

	if err := s.db.Save(payment).Error; err != nil {
		return fmt.Errorf("failed to save payment: %w", err)
	}

	return nil
}

// RefundPayment выполняет возврат платежа
func (s *PaymentService) RefundPayment(ctx context.Context, payment *models.Payment, amount float64) error {
	provider, exists := s.providers[payment.ProviderType]
	if !exists {
		return fmt.Errorf("unsupported payment provider: %s", payment.ProviderType)
	}

	if err := provider.RefundPayment(ctx, payment.TransactionID, amount); err != nil {
		return fmt.Errorf("failed to refund payment: %w", err)
	}

	// Создаем запись о возврате
	refund := &models.Refund{
		ID:            uuid.New().String(),
		PaymentID:     payment.ID,
		Amount:        amount,
		Currency:      payment.Currency,
		Status:        models.RefundStatusCompleted,
		RefundedAt:    time.Now(),
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	if err := s.db.Create(refund).Error; err != nil {
		return fmt.Errorf("failed to save refund: %w", err)
	}

	// Обновляем статус платежа
	payment.Status = models.PaymentStatusRefunded
	payment.UpdatedAt = time.Now()
	
	if err := s.db.Save(payment).Error; err != nil {
		return fmt.Errorf("failed to update payment status: %w", err)
	}

	return nil
}

// HandleWebhook обрабатывает вебхуки от платежных провайдеров
func (s *PaymentService) HandleWebhook(ctx context.Context, providerType payment.ProviderType, payload []byte, signature string) error {
	provider, exists := s.providers[providerType]
	if !exists {
		return fmt.Errorf("unsupported payment provider: %s", providerType)
	}

	// Проверяем и обрабатываем вебхук
	event, err := provider.ValidateWebhook(payload, signature)
	if err != nil {
		return fmt.Errorf("failed to validate webhook: %w", err)
	}

	// Находим платеж по TransactionID
	var payment models.Payment
	if err := s.db.Where("transaction_id = ?", event.TransactionID).First(&payment).Error; err != nil {
		return fmt.Errorf("payment not found: %w", err)
	}

	// Обновляем статус платежа
	payment.Status = event.Status
	payment.UpdatedAt = time.Now()
	payment.PaymentDetails = event.PaymentDetails

	if err := s.db.Save(&payment).Error; err != nil {
		return fmt.Errorf("failed to update payment: %w", err)
	}

	// Отправляем уведомление об изменении статуса
	if err := s.asyncService.PublishPaymentStatus(ctx, &models.PaymentStatusMessage{
		OrderID:   payment.OrderID,
		Status:    payment.Status,
		UpdatedAt: payment.UpdatedAt,
	}); err != nil {
		return fmt.Errorf("failed to publish status update: %w", err)
	}

	return nil
}

// GetPaymentStatus получает актуальный статус платежа от провайдера
func (s *PaymentService) GetPaymentStatus(ctx context.Context, payment *models.Payment) (models.PaymentStatus, error) {
	provider, exists := s.providers[payment.ProviderType]
	if !exists {
		return models.PaymentStatusUnknown, fmt.Errorf("unsupported payment provider: %s", payment.ProviderType)
	}

	status, err := provider.GetPaymentStatus(ctx, payment.TransactionID)
	if err != nil {
		return models.PaymentStatusUnknown, fmt.Errorf("failed to get payment status: %w", err)
	}

	// Обновляем статус в базе данных, если он изменился
	if status != payment.Status {
		payment.Status = status
		payment.UpdatedAt = time.Now()
		
		if err := s.db.Save(payment).Error; err != nil {
			return status, fmt.Errorf("failed to update payment status: %w", err)
		}

		// Публикуем обновление статуса
		if err := s.asyncService.PublishPaymentStatus(ctx, &models.PaymentStatusMessage{
			OrderID:   payment.OrderID,
			Status:    payment.Status,
			UpdatedAt: payment.UpdatedAt,
		}); err != nil {
			return status, fmt.Errorf("failed to publish status update: %w", err)
		}
	}

	return status, nil
}
