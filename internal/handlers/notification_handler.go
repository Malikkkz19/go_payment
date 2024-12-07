package handlers

import (
	"encoding/json"
	"go_payment/internal/models"
	"go_payment/internal/service"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// NotificationHandler представляет обработчик для уведомлений
type NotificationHandler struct {
	notificationService *service.NotificationService
}

// NewNotificationHandler создает новый обработчик уведомлений
func NewNotificationHandler(notificationService *service.NotificationService) *NotificationHandler {
	return &NotificationHandler{
		notificationService: notificationService,
	}
}

// RegisterRoutes регистрирует маршруты для уведомлений
func (h *NotificationHandler) RegisterRoutes(router *gin.Engine) {
	notifications := router.Group("/api/v1/notifications")
	{
		notifications.POST("/send", h.SendNotification)
		notifications.GET("/preferences/:user_id", h.GetPreferences)
		notifications.PUT("/preferences/:user_id", h.UpdatePreferences)
		notifications.GET("/status/:notification_id", h.GetStatus)
	}
}

// SendNotificationRequest представляет запрос на отправку уведомления
type SendNotificationRequest struct {
	TemplateID string                 `json:"template_id" binding:"required"`
	Recipient  string                 `json:"recipient" binding:"required"`
	Data       map[string]interface{} `json:"data"`
	Schedule   *time.Time             `json:"schedule,omitempty"`
}

// SendNotification обрабатывает запрос на отправку уведомления
func (h *NotificationHandler) SendNotification(c *gin.Context) {
	var req SendNotificationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	msg, err := h.notificationService.CreateNotification(c.Request.Context(), req.TemplateID, req.Recipient, req.Data)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if req.Schedule != nil {
		msg.ScheduledAt = req.Schedule
	}

	// В реальном приложении здесь нужно сохранить уведомление в базу
	// и отправить его в очередь для асинхронной обработки

	c.JSON(http.StatusAccepted, msg)
}

// GetPreferences возвращает настройки уведомлений пользователя
func (h *NotificationHandler) GetPreferences(c *gin.Context) {
	userID := c.Param("user_id")
	
	prefs, err := h.notificationService.GetUserPreferences(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, prefs)
}

// UpdatePreferences обновляет настройки уведомлений пользователя
func (h *NotificationHandler) UpdatePreferences(c *gin.Context) {
	userID := c.Param("user_id")

	var prefs models.NotificationPreferences
	if err := c.ShouldBindJSON(&prefs); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	prefs.UserID = userID
	prefs.UpdatedAt = time.Now()

	if err := h.notificationService.UpdateUserPreferences(c.Request.Context(), &prefs); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, prefs)
}

// GetStatus возвращает статус уведомления
func (h *NotificationHandler) GetStatus(c *gin.Context) {
	notificationID := c.Param("notification_id")

	// В реальном приложении здесь нужно получить статус из базы данных
	status := models.NotificationMessage{
		ID:     notificationID,
		Status: models.NotificationStatusSent,
	}

	c.JSON(http.StatusOK, status)
}
