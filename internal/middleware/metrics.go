package middleware

import (
	"go_payment/internal/metrics"
	"time"

	"github.com/gin-gonic/gin"
)

func MetricsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		// Увеличиваем счетчик активных соединений
		metrics.ActiveConnections.Inc()
		defer metrics.ActiveConnections.Dec()

		// Передаем управление следующему обработчику
		c.Next()

		// Собираем метрики после обработки запроса
		duration := time.Since(start).Seconds()

		// Записываем длительность запроса
		metrics.PaymentDuration.WithLabelValues(
			c.GetString("payment_provider"),
		).Observe(duration)

		// Если произошла ошибка, увеличиваем счетчик ошибок
		if len(c.Errors) > 0 {
			metrics.ErrorsTotal.WithLabelValues("http").Inc()
		}
	}
}
