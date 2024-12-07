package main

import (
	"fmt"
	pb "go_payment/api/proto/payment/v1"
	grpcServer "go_payment/internal/grpc"
	"go_payment/internal/service"
	"log"
	"net"

	"github.com/spf13/viper"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
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
	dsn := "host=localhost user=postgres password=postgres dbname=payment_service port=5432 sslmode=disable"
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	// Инициализация сервисов
	paymentService := service.NewPaymentService(db)

	// Настройка gRPC сервера
	port := viper.GetInt("grpc.port")
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	s := grpc.NewServer()
	pb.RegisterPaymentServiceServer(s, grpcServer.NewPaymentServer(paymentService))

	// Включаем reflection для удобства разработки
	reflection.Register(s)

	log.Printf("Starting gRPC server on port %d", port)
	if err := s.Serve(lis); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}
