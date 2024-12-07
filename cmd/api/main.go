package main

import (
	"go_payment/internal/handlers"
	"go_payment/internal/messaging"
	"go_payment/internal/middleware"
	"go_payment/internal/models"
	"go_payment/internal/service"
	"log"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/spf13/viper"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func main() {
	// Загрузка конфигурации
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("./configs")
	if err := viper.ReadInConfig(); err != nil {
		log.Fatalf("Error reading config file: %s", err)
	}

	// Подключение к базе данных
	dsn := viper.GetString("database.url")
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	// Автомиграция моделей
	err = db.AutoMigrate(&models.User{}, &models.Permission{}, &models.RolePermission{})
	if err != nil {
		log.Fatalf("Failed to migrate database: %v", err)
	}

	// Подключение к RabbitMQ
	rabbitmq, err := messaging.NewRabbitMQ(viper.GetString("rabbitmq.url"))
	if err != nil {
		log.Fatalf("Failed to connect to RabbitMQ: %v", err)
	}
	defer rabbitmq.Close()

	// Инициализация сервисов
	jwtSecret := viper.GetString("jwt.secret")
	if jwtSecret == "" {
		jwtSecret = "your-secret-key" // Для разработки, в продакшене использовать безопасный ключ
	}

	authService := service.NewAuthService(db, jwtSecret)
	asyncService := service.NewAsyncService(db, rabbitmq)
	paymentService := service.NewPaymentService(db, asyncService)

	// Инициализация сервиса уведомлений
	notificationService := service.NewNotificationService(
		viper.GetString("smtp.host"),
		viper.GetInt("smtp.port"),
		viper.GetString("smtp.user"),
		viper.GetString("smtp.password"),
	)

	// Запуск обработчика уведомлений
	err = rabbitmq.ConsumeNotifications(notificationService.ProcessNotification)
	if err != nil {
		log.Fatalf("Failed to start notification consumer: %v", err)
	}

	// Запуск обработки асинхронных операций
	if err := asyncService.StartProcessing(); err != nil {
		log.Fatalf("Failed to start async processing: %v", err)
	}

	// Настройка Gin
	r := gin.Default()

	// Добавляем middleware для метрик
	r.Use(middleware.MetricsMiddleware())

	// Публичные endpoints
	r.GET("/metrics", gin.WrapH(promhttp.Handler()))
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "UP"})
	})

	// Аутентификация
	auth := r.Group("/auth")
	{
		authHandler := handlers.NewAuthHandler(authService)
		auth.POST("/register", authHandler.Register)
		auth.POST("/login", authHandler.Login)
		auth.POST("/refresh", authHandler.RefreshToken)
	}

	// Защищенные endpoints
	api := r.Group("/api/v1")
	api.Use(middleware.AuthMiddleware(authService))
	{
		// Endpoints для платежей (доступны только для админов и менеджеров)
		payments := api.Group("/payments")
		payments.Use(middleware.RoleMiddleware(models.RoleAdmin, models.RoleManager))
		{
			paymentHandler := handlers.NewPaymentHandler(paymentService)
			payments.POST("/", paymentHandler.CreatePayment)
			payments.GET("/:orderID", paymentHandler.GetPayment)
			// Возврат средств доступен только для админов
			payments.POST("/:orderID/refund", middleware.RoleMiddleware(models.RoleAdmin), paymentHandler.RefundPayment)
		}

		// Endpoints для уведомлений
		notifications := api.Group("/notifications")
		{
			notificationHandler := handlers.NewNotificationHandler(notificationService)
			notifications.POST("/", notificationHandler.SendNotification)
		}

		// Endpoints для пользователей (только для админов)
		users := api.Group("/users")
		users.Use(middleware.RoleMiddleware(models.RoleAdmin))
		{
			// TODO: добавить управление пользователями
		}
	}

	// Запуск сервера
	port := viper.GetString("server.port")
	if err := r.Run(":" + port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
