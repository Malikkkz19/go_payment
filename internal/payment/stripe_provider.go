package payment

import (
	"context"
	"encoding/json"
	"fmt"
	"go_payment/internal/models"
	"log"

	"github.com/stripe/stripe-go/v74"
	"github.com/stripe/stripe-go/v74/charge"
	"github.com/stripe/stripe-go/v74/refund"
	"github.com/stripe/stripe-go/v74/webhook"
)

// StripeProvider реализует интерфейс Provider для Stripe
type StripeProvider struct {
	secretKey    string
	webhookKey   string
	endpointURL  string
	testMode     bool
}

// NewStripeProvider создает новый экземпляр провайдера Stripe
func NewStripeProvider() *StripeProvider {
	return &StripeProvider{}
}

// Initialize инициализирует провайдер с конфигурацией
func (p *StripeProvider) Initialize(config map[string]string) error {
	p.secretKey = config["secretKey"]
	p.webhookKey = config["webhookKey"]
	p.endpointURL = config["endpointURL"]
	p.testMode = config["testMode"] == "true"

	stripe.Key = p.secretKey
	return nil
}

// ProcessPayment обрабатывает платеж через Stripe
func (p *StripeProvider) ProcessPayment(ctx context.Context, req PaymentRequest) (*PaymentResponse, error) {
	// Конвертируем сумму в центы (Stripe работает с наименьшими единицами валюты)
	amountInCents := int64(req.Amount * 100)

	// Создаем параметры для создания платежа
	params := &stripe.ChargeParams{
		Amount:      stripe.Int64(amountInCents),
		Currency:    stripe.String(string(req.Currency)),
		Description: stripe.String(req.Description),
		Metadata: map[string]string{
			"order_id": req.OrderID,
		},
	}

	// Добавляем информацию о клиенте, если есть
	if req.CustomerEmail != "" {
		params.ReceiptEmail = stripe.String(req.CustomerEmail)
	}

	// Добавляем дополнительные метаданные
	for k, v := range req.MetaData {
		if strVal, ok := v.(string); ok {
			params.Metadata[k] = strVal
		}
	}

	// Создаем платеж
	charge, err := charge.New(params)
	if err != nil {
		return &PaymentResponse{
			Success:      false,
			ErrorMessage: err.Error(),
			Status:      models.PaymentStatusFailed,
		}, fmt.Errorf("failed to create stripe charge: %w", err)
	}

	// Определяем статус платежа
	status := models.PaymentStatusPending
	if charge.Paid {
		status = models.PaymentStatusCompleted
	} else if charge.Status == "failed" {
		status = models.PaymentStatusFailed
	}

	// Формируем детали платежа
	details := map[string]interface{}{
		"charge_id":          charge.ID,
		"payment_method":     charge.PaymentMethod,
		"receipt_url":        charge.ReceiptURL,
		"statement_descriptor": charge.StatementDescriptor,
		"risk_level":         charge.Outcome.RiskLevel,
		"seller_message":     charge.Outcome.SellerMessage,
	}

	return &PaymentResponse{
		Success:       charge.Paid,
		TransactionID: charge.ID,
		Status:       status,
		PaymentDetails: details,
	}, nil
}

// ValidateWebhook проверяет и обрабатывает вебхук от Stripe
func (p *StripeProvider) ValidateWebhook(payload []byte, signature string) (*WebhookEvent, error) {
	event, err := webhook.ConstructEvent(payload, signature, p.webhookKey)
	if err != nil {
		return nil, fmt.Errorf("failed to verify webhook signature: %w", err)
	}

	var status models.PaymentStatus
	var amount float64
	var currency string
	var transactionID string
	details := make(map[string]interface{})

	switch event.Type {
	case "charge.succeeded":
		var charge stripe.Charge
		err := json.Unmarshal(event.Data.Raw, &charge)
		if err != nil {
			return nil, fmt.Errorf("failed to parse charge data: %w", err)
		}
		status = models.PaymentStatusCompleted
		amount = float64(charge.Amount) / 100
		currency = string(charge.Currency)
		transactionID = charge.ID
		details["receipt_url"] = charge.ReceiptURL
		details["payment_method"] = charge.PaymentMethod

	case "charge.failed":
		var charge stripe.Charge
		err := json.Unmarshal(event.Data.Raw, &charge)
		if err != nil {
			return nil, fmt.Errorf("failed to parse charge data: %w", err)
		}
		status = models.PaymentStatusFailed
		amount = float64(charge.Amount) / 100
		currency = string(charge.Currency)
		transactionID = charge.ID
		details["failure_code"] = charge.FailureCode
		details["failure_message"] = charge.FailureMessage

	case "charge.refunded":
		var charge stripe.Charge
		err := json.Unmarshal(event.Data.Raw, &charge)
		if err != nil {
			return nil, fmt.Errorf("failed to parse charge data: %w", err)
		}
		status = models.PaymentStatusRefunded
		amount = float64(charge.AmountRefunded) / 100
		currency = string(charge.Currency)
		transactionID = charge.ID
		details["refund_reason"] = charge.RefundReason
	}

	return &WebhookEvent{
		Type:          event.Type,
		TransactionID: transactionID,
		Status:       status,
		Amount:       amount,
		Currency:     currency,
		PaymentDetails: details,
	}, nil
}

// RefundPayment выполняет возврат платежа
func (p *StripeProvider) RefundPayment(ctx context.Context, transactionID string, amount float64) error {
	amountInCents := int64(amount * 100)

	params := &stripe.RefundParams{
		Charge: stripe.String(transactionID),
		Amount: stripe.Int64(amountInCents),
	}

	_, err := refund.New(params)
	if err != nil {
		return fmt.Errorf("failed to create refund: %w", err)
	}

	return nil
}

// GetPaymentStatus получает текущий статус платежа
func (p *StripeProvider) GetPaymentStatus(ctx context.Context, transactionID string) (models.PaymentStatus, error) {
	ch, err := charge.Get(transactionID, nil)
	if err != nil {
		return models.PaymentStatusUnknown, fmt.Errorf("failed to get charge: %w", err)
	}

	switch {
	case ch.Refunded:
		return models.PaymentStatusRefunded, nil
	case ch.Paid:
		return models.PaymentStatusCompleted, nil
	case ch.Status == "failed":
		return models.PaymentStatusFailed, nil
	default:
		return models.PaymentStatusPending, nil
	}
}
