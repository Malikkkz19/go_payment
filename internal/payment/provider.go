package payment

import (
	"context"
	"go_payment/internal/models"
)

type PaymentRequest struct {
	OrderID       string
	Amount        float64
	Currency      string
	CustomerID    string
	CustomerEmail string
	Description   string
	MetaData      map[string]interface{}
}

type PaymentResponse struct {
	Success        bool
	TransactionID  string
	Status         models.PaymentStatus
	ErrorMessage   string
	PaymentDetails map[string]interface{}
}

type Provider interface {
	Initialize(config map[string]string) error
	ProcessPayment(ctx context.Context, req PaymentRequest) (*PaymentResponse, error)
	ValidateWebhook(payload []byte, signature string) (*WebhookEvent, error)
	RefundPayment(ctx context.Context, transactionID string, amount float64) error
	GetPaymentStatus(ctx context.Context, transactionID string) (models.PaymentStatus, error)
}

type WebhookEvent struct {
	Type           string
	TransactionID  string
	Status         models.PaymentStatus
	Amount         float64
	Currency       string
	PaymentDetails map[string]interface{}
}
