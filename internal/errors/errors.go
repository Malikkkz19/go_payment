package errors

import (
	"fmt"
	"time"
)

// ErrorType представляет тип ошибки
type ErrorType string

const (
	// Типы ошибок
	ErrorTypeValidation    ErrorType = "validation"
	ErrorTypeDatabase      ErrorType = "database"
	ErrorTypePayment       ErrorType = "payment"
	ErrorTypeMessaging     ErrorType = "messaging"
	ErrorTypeInternal      ErrorType = "internal"
	ErrorTypeAuthentication ErrorType = "authentication"
)

// RetryStrategy определяет стратегию повторных попыток
type RetryStrategy struct {
	MaxAttempts     int           // Максимальное количество попыток
	InitialInterval time.Duration // Начальный интервал между попытками
	MaxInterval     time.Duration // Максимальный интервал между попытками
	Multiplier      float64       // Множитель для увеличения интервала
}

// PaymentError представляет ошибку обработки платежа
type PaymentError struct {
	Type        ErrorType // Тип ошибки
	Code        string    // Код ошибки
	Message     string    // Сообщение об ошибке
	Retryable   bool      // Можно ли повторить операцию
	OrderID     string    // ID заказа
	ProviderErr error     // Исходная ошибка от провайдера
}

func (e *PaymentError) Error() string {
	if e.ProviderErr != nil {
		return fmt.Sprintf("[%s] %s: %s (OrderID: %s) - %v", e.Type, e.Code, e.Message, e.OrderID, e.ProviderErr)
	}
	return fmt.Sprintf("[%s] %s: %s (OrderID: %s)", e.Type, e.Code, e.Message, e.OrderID)
}

// NewPaymentError создает новую ошибку платежа
func NewPaymentError(errType ErrorType, code, message, orderID string, retryable bool, providerErr error) *PaymentError {
	return &PaymentError{
		Type:        errType,
		Code:        code,
		Message:     message,
		Retryable:   retryable,
		OrderID:     orderID,
		ProviderErr: providerErr,
	}
}

// DefaultRetryStrategy возвращает стратегию по умолчанию
func DefaultRetryStrategy() *RetryStrategy {
	return &RetryStrategy{
		MaxAttempts:     3,
		InitialInterval: 1 * time.Second,
		MaxInterval:     30 * time.Second,
		Multiplier:      2.0,
	}
}

// CalculateNextInterval вычисляет следующий интервал для повторной попытки
func (s *RetryStrategy) CalculateNextInterval(attempt int) time.Duration {
	if attempt <= 0 {
		return s.InitialInterval
	}

	interval := float64(s.InitialInterval) * float64(s.Multiplier)
	maxInterval := float64(s.MaxInterval)

	if interval > maxInterval {
		return s.MaxInterval
	}

	return time.Duration(interval)
}

// ShouldRetry определяет, нужно ли повторить попытку
func (s *RetryStrategy) ShouldRetry(attempt int, err error) bool {
	if attempt >= s.MaxAttempts {
		return false
	}

	if paymentErr, ok := err.(*PaymentError); ok {
		return paymentErr.Retryable
	}

	return false
}
