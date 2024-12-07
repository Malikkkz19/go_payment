package handlers

import (
	"go_payment/internal/models"
	"go_payment/internal/payment"
	"go_payment/internal/service"
	"net/http"

	"github.com/gin-gonic/gin"
)

type PaymentHandler struct {
	paymentService *service.PaymentService
}

func NewPaymentHandler(paymentService *service.PaymentService) *PaymentHandler {
	return &PaymentHandler{
		paymentService: paymentService,
	}
}

type CreatePaymentRequest struct {
	OrderID       string                 `json:"order_id" binding:"required"`
	Amount        float64                `json:"amount" binding:"required"`
	Currency      string                 `json:"currency" binding:"required"`
	Provider      string                 `json:"provider" binding:"required"`
	CustomerID    string                 `json:"customer_id" binding:"required"`
	CustomerEmail string                 `json:"customer_email" binding:"required"`
	Description   string                 `json:"description"`
	MetaData      map[string]interface{} `json:"metadata"`
}

func (h *PaymentHandler) CreatePayment(c *gin.Context) {
	var req CreatePaymentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	paymentReq := payment.PaymentRequest{
		OrderID:       req.OrderID,
		Amount:        req.Amount,
		Currency:      req.Currency,
		CustomerID:    req.CustomerID,
		CustomerEmail: req.CustomerEmail,
		Description:   req.Description,
		MetaData:      req.MetaData,
	}

	provider := models.PaymentProvider(req.Provider)
	payment, err := h.paymentService.ProcessPayment(c.Request.Context(), paymentReq, provider)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, payment)
}

func (h *PaymentHandler) GetPayment(c *gin.Context) {
	orderID := c.Param("orderID")
	payment, err := h.paymentService.GetPayment(orderID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "payment not found"})
		return
	}

	c.JSON(http.StatusOK, payment)
}

func (h *PaymentHandler) RefundPayment(c *gin.Context) {
	orderID := c.Param("orderID")
	if err := h.paymentService.RefundPayment(c.Request.Context(), orderID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "payment refunded successfully"})
}
