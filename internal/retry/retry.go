package retry

import (
	"context"
	"go_payment/internal/errors"
	"log"
	"time"
)

// Operation представляет операцию, которую нужно повторить
type Operation func(ctx context.Context) error

// WithRetry выполняет операцию с повторными попытками
func WithRetry(ctx context.Context, op Operation, strategy *errors.RetryStrategy) error {
	var lastErr error

	for attempt := 0; attempt < strategy.MaxAttempts; attempt++ {
		// Проверяем контекст перед каждой попыткой
		if ctx.Err() != nil {
			return ctx.Err()
		}

		// Выполняем операцию
		err := op(ctx)
		if err == nil {
			return nil
		}

		lastErr = err

		// Проверяем, нужно ли повторять попытку
		if !strategy.ShouldRetry(attempt, err) {
			log.Printf("Not retrying operation after attempt %d: %v", attempt+1, err)
			return err
		}

		// Вычисляем интервал для следующей попытки
		interval := strategy.CalculateNextInterval(attempt)

		log.Printf("Retrying operation after attempt %d in %v: %v", attempt+1, interval, err)

		// Ждем перед следующей попыткой
		timer := time.NewTimer(interval)
		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		case <-timer.C:
			continue
		}
	}

	return lastErr
}

// WithRetryBackoff выполняет операцию с экспоненциальной задержкой
func WithRetryBackoff(ctx context.Context, op Operation) error {
	return WithRetry(ctx, op, errors.DefaultRetryStrategy())
}
