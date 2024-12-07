package payment

import (
	"context"
	"encoding/json"
	"fmt"
	"go_payment/internal/models"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/plutov/paypal/v4"
)

// PayPalProvider реализует интерфейс Provider для PayPal
type PayPalProvider struct {
	client      *paypal.Client
	clientID    string
	secretKey   string
	webhookID   string
	endpointURL string
	testMode    bool
}

// NewPayPalProvider создает новый экземпляр провайдера PayPal
func NewPayPalProvider() *PayPalProvider {
	return &PayPalProvider{}
}

// Initialize инициализирует провайдер с конфигурацией
func (p *PayPalProvider) Initialize(config map[string]string) error {
	p.clientID = config["clientID"]
	p.secretKey = config["secretKey"]
	p.webhookID = config["webhookID"]
	p.endpointURL = config["endpointURL"]
	p.testMode = config["testMode"] == "true"

	// Определяем API URL в зависимости от режима
	apiBase := paypal.APIBaseSandBox
	if !p.testMode {
		apiBase = paypal.APIBaseLive
	}

	// Создаем клиент PayPal
	client, err := paypal.NewClient(p.clientID, p.secretKey, apiBase)
	if err != nil {
		return fmt.Errorf("failed to create PayPal client: %w", err)
	}

	// Получаем токен доступа
	_, err = client.GetAccessToken(context.Background())
	if err != nil {
		return fmt.Errorf("failed to get PayPal access token: %w", err)
	}

	p.client = client
	return nil
}

// ProcessPayment обрабатывает платеж через PayPal
func (p *PayPalProvider) ProcessPayment(ctx context.Context, req PaymentRequest) (*PaymentResponse, error) {
	// Создаем заказ PayPal
	order, err := p.client.CreateOrder(ctx, paypal.OrderIntentCapture, []paypal.PurchaseUnitRequest{
		{
			ReferenceID: req.OrderID,
			Amount: &paypal.Money{
				Currency: req.Currency,
				Value:    fmt.Sprintf("%.2f", req.Amount),
			},
			Description: req.Description,
			CustomID:    req.OrderID,
		},
	})

	if err != nil {
		return &PaymentResponse{
			Success:      false,
			ErrorMessage: err.Error(),
			Status:      models.PaymentStatusFailed,
		}, fmt.Errorf("failed to create PayPal order: %w", err)
	}

	// Захватываем платеж
	capture, err := p.client.CaptureOrder(ctx, order.ID)
	if err != nil {
		return &PaymentResponse{
			Success:      false,
			ErrorMessage: err.Error(),
			Status:      models.PaymentStatusFailed,
		}, fmt.Errorf("failed to capture PayPal payment: %w", err)
	}

	// Определяем статус платежа
	status := models.PaymentStatusPending
	if capture.Status == "COMPLETED" {
		status = models.PaymentStatusCompleted
	} else if capture.Status == "DECLINED" {
		status = models.PaymentStatusFailed
	}

	// Формируем детали платежа
	details := map[string]interface{}{
		"order_id":        order.ID,
		"capture_id":      capture.ID,
		"payment_source":  capture.PaymentSource,
		"status":          capture.Status,
		"create_time":     capture.CreateTime,
		"update_time":     capture.UpdateTime,
	}

	return &PaymentResponse{
		Success:        status == models.PaymentStatusCompleted,
		TransactionID:  capture.ID,
		Status:        status,
		PaymentDetails: details,
	}, nil
}

// ValidateWebhook проверяет и обрабатывает вебхук от PayPal
func (p *PayPalProvider) ValidateWebhook(payload []byte, signature string) (*WebhookEvent, error) {
	// Проверяем подпись вебхука
	headers := http.Header{}
	headers.Set("Paypal-Transmission-Sig", signature)
	valid, err := p.client.ValidateWebhookSignature(context.Background(), payload, headers, p.webhookID)
	if err != nil || !valid {
		return nil, fmt.Errorf("invalid webhook signature")
	}

	// Разбираем payload
	var event paypal.WebhookEvent
	if err := json.Unmarshal(payload, &event); err != nil {
		return nil, fmt.Errorf("failed to parse webhook payload: %w", err)
	}

	var status models.PaymentStatus
	var amount float64
	var currency string
	var transactionID string
	details := make(map[string]interface{})

	// Обрабатываем различные типы событий
	switch event.ResourceType {
	case "capture":
		var capture paypal.CaptureResource
		if err := json.Unmarshal(event.Resource, &capture); err != nil {
			return nil, fmt.Errorf("failed to parse capture data: %w", err)
		}

		amount, _ = capture.Amount.Value.Float64()
		currency = capture.Amount.Currency
		transactionID = capture.ID

		switch event.EventType {
		case "PAYMENT.CAPTURE.COMPLETED":
			status = models.PaymentStatusCompleted
		case "PAYMENT.CAPTURE.DENIED":
			status = models.PaymentStatusFailed
		case "PAYMENT.CAPTURE.REFUNDED":
			status = models.PaymentStatusRefunded
		}

		details["capture_id"] = capture.ID
		details["status"] = capture.Status
		details["create_time"] = capture.CreateTime
		details["update_time"] = capture.UpdateTime
	}

	return &WebhookEvent{
		Type:           event.EventType,
		TransactionID:  transactionID,
		Status:        status,
		Amount:        amount,
		Currency:      currency,
		PaymentDetails: details,
	}, nil
}

// RefundPayment выполняет возврат платежа
func (p *PayPalProvider) RefundPayment(ctx context.Context, transactionID string, amount float64) error {
	refundRequest := paypal.RefundRequest{
		Amount: &paypal.Money{
			Value:    fmt.Sprintf("%.2f", amount),
			Currency: "USD", // Здесь можно добавить параметр для указания валюты
		},
		NoteToPayer: "Refund for order",
	}

	_, err := p.client.RefundCapture(ctx, transactionID, refundRequest)
	if err != nil {
		return fmt.Errorf("failed to refund PayPal payment: %w", err)
	}

	return nil
}

// GetPaymentStatus получает текущий статус платежа
func (p *PayPalProvider) GetPaymentStatus(ctx context.Context, transactionID string) (models.PaymentStatus, error) {
	capture, err := p.client.GetCapture(ctx, transactionID)
	if err != nil {
		return models.PaymentStatusUnknown, fmt.Errorf("failed to get PayPal capture: %w", err)
	}

	switch capture.Status {
	case "COMPLETED":
		return models.PaymentStatusCompleted, nil
	case "DECLINED":
		return models.PaymentStatusFailed, nil
	case "REFUNDED":
		return models.PaymentStatusRefunded, nil
	default:
		return models.PaymentStatusPending, nil
	}
}
