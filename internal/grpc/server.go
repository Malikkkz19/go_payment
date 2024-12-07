package grpc

import (
	"context"
	pb "go_payment/api/proto/payment/v1"
	"go_payment/internal/models"
	"go_payment/internal/service"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type PaymentServer struct {
	pb.UnimplementedPaymentServiceServer
	paymentService *service.PaymentService
}

func NewPaymentServer(paymentService *service.PaymentService) *PaymentServer {
	return &PaymentServer{
		paymentService: paymentService,
	}
}

func (s *PaymentServer) CreatePayment(ctx context.Context, req *pb.CreatePaymentRequest) (*pb.CreatePaymentResponse, error) {
	// Конвертируем gRPC запрос в модель платежа
	payment := &models.Payment{
		OrderID:       req.OrderId,
		Amount:        req.Amount,
		Currency:      req.Currency,
		CustomerID:    req.CustomerId,
		CustomerEmail: req.CustomerEmail,
		Description:   req.Description,
		MetaData:      convertMetadataToJSON(req.Metadata),
		Provider:      models.PaymentProvider(req.Provider.String()),
	}

	// Обрабатываем платеж через сервис
	result, err := s.paymentService.ProcessPayment(ctx, payment)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to process payment: %v", err)
	}

	// Конвертируем результат обратно в gRPC ответ
	return &pb.CreatePaymentResponse{
		Payment: convertPaymentToProto(result),
	}, nil
}

func (s *PaymentServer) GetPayment(ctx context.Context, req *pb.GetPaymentRequest) (*pb.GetPaymentResponse, error) {
	payment, err := s.paymentService.GetPayment(req.OrderId)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "payment not found: %v", err)
	}

	return &pb.GetPaymentResponse{
		Payment: convertPaymentToProto(payment),
	}, nil
}

func (s *PaymentServer) RefundPayment(ctx context.Context, req *pb.RefundPaymentRequest) (*pb.RefundPaymentResponse, error) {
	err := s.paymentService.RefundPayment(ctx, req.OrderId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to refund payment: %v", err)
	}

	return &pb.RefundPaymentResponse{
		RefundId:   "ref_" + req.OrderId, // В реальном приложении генерировать уникальный ID
		Status:     pb.PaymentStatus_PAYMENT_STATUS_SUCCESS,
		RefundDate: timestamppb.Now(),
	}, nil
}

func (s *PaymentServer) ListPayments(ctx context.Context, req *pb.ListPaymentsRequest) (*pb.ListPaymentsResponse, error) {
	// TODO: Реализовать получение списка платежей с фильтрацией
	return nil, status.Errorf(codes.Unimplemented, "method ListPayments not implemented")
}

// Вспомогательные функции для конвертации типов
func convertPaymentToProto(p *models.Payment) *pb.Payment {
	return &pb.Payment{
		OrderId:       p.OrderID,
		Amount:        p.Amount,
		Currency:      p.Currency,
		Status:        convertStatusToProto(p.Status),
		Provider:      convertProviderToProto(p.Provider),
		ProviderTxnId: p.ProviderTxnID,
		CustomerId:    p.CustomerID,
		CustomerEmail: p.CustomerEmail,
		Description:   p.Description,
		ErrorMessage:  p.ErrorMessage,
		Metadata:      convertJSONToMetadata(p.MetaData),
		PaymentDate:   timestamppb.New(p.PaymentDate),
	}
}

func convertStatusToProto(status models.PaymentStatus) pb.PaymentStatus {
	switch status {
	case models.PaymentStatusPending:
		return pb.PaymentStatus_PAYMENT_STATUS_PENDING
	case models.PaymentStatusSuccess:
		return pb.PaymentStatus_PAYMENT_STATUS_SUCCESS
	case models.PaymentStatusFailed:
		return pb.PaymentStatus_PAYMENT_STATUS_FAILED
	case models.PaymentStatusCancelled:
		return pb.PaymentStatus_PAYMENT_STATUS_CANCELLED
	default:
		return pb.PaymentStatus_PAYMENT_STATUS_UNSPECIFIED
	}
}

func convertProviderToProto(provider models.PaymentProvider) pb.PaymentProvider {
	switch provider {
	case models.PaymentProviderStripe:
		return pb.PaymentProvider_PAYMENT_PROVIDER_STRIPE
	case models.PaymentProviderPayPal:
		return pb.PaymentProvider_PAYMENT_PROVIDER_PAYPAL
	default:
		return pb.PaymentProvider_PAYMENT_PROVIDER_UNSPECIFIED
	}
}

func convertMetadataToJSON(metadata map[string]string) models.JSON {
	result := make(models.JSON)
	for k, v := range metadata {
		result[k] = v
	}
	return result
}

func convertJSONToMetadata(json models.JSON) map[string]string {
	result := make(map[string]string)
	for k, v := range json {
		if str, ok := v.(string); ok {
			result[k] = str
		}
	}
	return result
}
