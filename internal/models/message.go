package models

import "time"

// PaymentMessage представляет сообщение о платеже для очереди
type PaymentMessage struct {
	OrderID       string          `json:"order_id"`
	Amount        float64         `json:"amount"`
	Currency      string          `json:"currency"`
	Status        PaymentStatus   `json:"status"`
	Provider      PaymentProvider `json:"provider"`
	CustomerID    string          `json:"customer_id"`
	CustomerEmail string          `json:"customer_email"`
	CreatedAt     time.Time       `json:"created_at"`
	MetaData      JSON           `json:"metadata,omitempty"`
}

// PaymentStatusMessage представляет сообщение об изменении статуса платежа
type PaymentStatusMessage struct {
	OrderID     string        `json:"order_id"`
	OldStatus   PaymentStatus `json:"old_status"`
	NewStatus   PaymentStatus `json:"new_status"`
	UpdatedAt   time.Time     `json:"updated_at"`
	Description string        `json:"description,omitempty"`
}

// NotificationMessage представляет сообщение для уведомления
type NotificationMessage struct {
	Type      string    `json:"type"`      // email, sms, webhook
	Recipient string    `json:"recipient"` // email address, phone number, or webhook URL
	Subject   string    `json:"subject"`
	Content   string    `json:"content"`
	Metadata  JSON      `json:"metadata,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}
