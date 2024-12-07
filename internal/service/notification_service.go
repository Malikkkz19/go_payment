package service

import (
	"context"
	"encoding/json"
	"fmt"
	"go_payment/internal/metrics"
	"go_payment/internal/models"
	"html/template"
	"log"
	"time"

	"github.com/google/uuid"
	"gopkg.in/gomail.v2"
)

// NotificationService представляет сервис для работы с уведомлениями
type NotificationService struct {
	templates map[string]*template.Template
	mailer    *gomail.Dialer
	// Можно добавить другие провайдеры уведомлений (SMS, Push и т.д.)
}

// NewNotificationService создает новый экземпляр сервиса уведомлений
func NewNotificationService(smtpHost string, smtpPort int, smtpUser, smtpPassword string) *NotificationService {
	mailer := gomail.NewDialer(smtpHost, smtpPort, smtpUser, smtpPassword)
	
	return &NotificationService{
		templates: make(map[string]*template.Template),
		mailer:    mailer,
	}
}

// ProcessNotification обрабатывает уведомление
func (s *NotificationService) ProcessNotification(ctx context.Context, msg *models.NotificationMessage) error {
	done := metrics.TrackProcessingTime("notifications", string(msg.Type))
	defer done()

	log.Printf("Processing notification: %s of type %s for recipient %s", 
		msg.ID, msg.Type, msg.Recipient)

	var err error
	switch msg.Type {
	case models.NotificationTypeEmail:
		err = s.sendEmail(ctx, msg)
	case models.NotificationTypeSMS:
		err = s.sendSMS(ctx, msg)
	case models.NotificationTypePush:
		err = s.sendPushNotification(ctx, msg)
	case models.NotificationTypeWebhook:
		err = s.sendWebhook(ctx, msg)
	default:
		err = fmt.Errorf("unsupported notification type: %s", msg.Type)
	}

	if err != nil {
		metrics.RecordProcessingError("notifications", fmt.Sprintf("%s_error", msg.Type))
		return fmt.Errorf("failed to send notification: %w", err)
	}

	metrics.IncrementProcessedMessage("notifications", string(msg.Status))
	return nil
}

// CreateNotification создает новое уведомление
func (s *NotificationService) CreateNotification(ctx context.Context, templateID string, recipient string, data map[string]interface{}) (*models.NotificationMessage, error) {
	// Здесь должна быть логика получения шаблона из базы данных
	template := &models.NotificationTemplate{
		ID:      templateID,
		Type:    models.NotificationTypeEmail,
		Subject: "Payment Notification",
		Content: "Your payment has been processed",
	}

	msg := &models.NotificationMessage{
		ID:        uuid.New().String(),
		Type:      template.Type,
		Status:    models.NotificationStatusPending,
		Recipient: recipient,
		Subject:   template.Subject,
		Content:   template.Content,
		Metadata:  make(map[string]string),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Добавить метаданные из данных шаблона
	for k, v := range data {
		if str, ok := v.(string); ok {
			msg.Metadata[k] = str
		}
	}

	return msg, nil
}

// sendEmail отправляет email уведомление
func (s *NotificationService) sendEmail(ctx context.Context, msg *models.NotificationMessage) error {
	m := gomail.NewMessage()
	m.SetHeader("From", "noreply@payment-service.com")
	m.SetHeader("To", msg.Recipient)
	m.SetHeader("Subject", msg.Subject)
	m.SetBody("text/html", msg.Content)

	if err := s.mailer.DialAndSend(m); err != nil {
		return fmt.Errorf("failed to send email: %w", err)
	}

	now := time.Now()
	msg.Status = models.NotificationStatusSent
	msg.SentAt = &now
	msg.UpdatedAt = now

	return nil
}

// sendSMS отправляет SMS уведомление
func (s *NotificationService) sendSMS(ctx context.Context, msg *models.NotificationMessage) error {
	// Здесь должна быть реализация отправки SMS
	// Например, через Twilio или другой сервис
	return fmt.Errorf("SMS notifications not implemented yet")
}

// sendPushNotification отправляет push-уведомление
func (s *NotificationService) sendPushNotification(ctx context.Context, msg *models.NotificationMessage) error {
	// Здесь должна быть реализация отправки push-уведомлений
	// Например, через Firebase Cloud Messaging
	return fmt.Errorf("push notifications not implemented yet")
}

// sendWebhook отправляет webhook уведомление
func (s *NotificationService) sendWebhook(ctx context.Context, msg *models.NotificationMessage) error {
	// Здесь должна быть реализация отправки webhook
	// Например, через HTTP POST запрос
	return fmt.Errorf("webhook notifications not implemented yet")
}

// LoadTemplates загружает шаблоны уведомлений
func (s *NotificationService) LoadTemplates() error {
	// Здесь должна быть логика загрузки шаблонов из файлов или базы данных
	return nil
}

// GetUserPreferences получает настройки уведомлений пользователя
func (s *NotificationService) GetUserPreferences(ctx context.Context, userID string) (*models.NotificationPreferences, error) {
	// Здесь должна быть логика получения настроек из базы данных
	return &models.NotificationPreferences{
		UserID:   userID,
		Channels: []models.NotificationType{models.NotificationTypeEmail},
		Enabled:  true,
		UpdatedAt: time.Now(),
	}, nil
}

// UpdateUserPreferences обновляет настройки уведомлений пользователя
func (s *NotificationService) UpdateUserPreferences(ctx context.Context, prefs *models.NotificationPreferences) error {
	// Здесь должна быть логика сохранения настроек в базу данных
	return nil
}
