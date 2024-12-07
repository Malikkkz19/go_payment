package models

import (
	"database/sql/driver"
	"encoding/json"
	"go_payment/internal/payment"
	"time"

	"gorm.io/gorm"
)

// PaymentStatus определяет статус платежа
type PaymentStatus string

const (
	PaymentStatusPending   PaymentStatus = "pending"
	PaymentStatusCompleted PaymentStatus = "completed"
	PaymentStatusFailed    PaymentStatus = "failed"
	PaymentStatusCancelled PaymentStatus = "cancelled"
	PaymentStatusRefunded  PaymentStatus = "refunded"
	PaymentStatusUnknown   PaymentStatus = "unknown"
)

// JSON представляет JSON данные в базе данных
type JSON map[string]interface{}

// Value реализует интерфейс driver.Valuer для JSON
func (j JSON) Value() (driver.Value, error) {
	if j == nil {
		return nil, nil
	}
	return json.Marshal(j)
}

// Scan реализует интерфейс sql.Scanner для JSON
func (j *JSON) Scan(value interface{}) error {
	if value == nil {
		*j = nil
		return nil
	}
	return json.Unmarshal(value.([]byte), &j)
}

// Payment представляет платеж в системе
type Payment struct {
	gorm.Model
	ID             string                `json:"id" gorm:"primaryKey"`
	OrderID        string                `json:"order_id" gorm:"uniqueIndex"`
	CustomerID     string                `json:"customer_id"`
	CustomerEmail  string                `json:"customer_email"`
	Amount         float64               `json:"amount"`
	Currency       string                `json:"currency"`
	Description    string                `json:"description"`
	Status         PaymentStatus         `json:"status"`
	ProviderType   payment.ProviderType  `json:"provider_type"`
	TransactionID  string                `json:"transaction_id"`
	PaymentDetails JSON                  `json:"payment_details"`
	Metadata       JSON                  `json:"metadata"`
	ErrorMessage   string                `json:"error_message,omitempty"`
	CompletedAt    *time.Time            `json:"completed_at,omitempty"`
}

// Refund представляет возврат платежа
type Refund struct {
	gorm.Model
	ID            string       `json:"id" gorm:"primaryKey"`
	PaymentID     string       `json:"payment_id"`
	Amount        float64      `json:"amount"`
	Currency      string       `json:"currency"`
	Status        RefundStatus `json:"status"`
	Reason        string       `json:"reason,omitempty"`
	RefundedAt    time.Time    `json:"refunded_at"`
}

// RefundStatus определяет статус возврата
type RefundStatus string

const (
	RefundStatusPending   RefundStatus = "pending"
	RefundStatusCompleted RefundStatus = "completed"
	RefundStatusFailed    RefundStatus = "failed"
)

// PaymentStatusMessage представляет сообщение об изменении статуса платежа
type PaymentStatusMessage struct {
	OrderID      string        `json:"order_id"`
	OldStatus    PaymentStatus `json:"old_status"`
	NewStatus    PaymentStatus `json:"new_status"`
	TransactionID string       `json:"transaction_id,omitempty"`
	UpdatedAt    time.Time     `json:"updated_at"`
	Metadata     JSON          `json:"metadata,omitempty"`
}
