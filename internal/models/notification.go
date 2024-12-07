package models

import "time"

// NotificationType определяет тип уведомления
type NotificationType string

const (
	NotificationTypeEmail   NotificationType = "email"
	NotificationTypeSMS     NotificationType = "sms"
	NotificationTypePush    NotificationType = "push"
	NotificationTypeWebhook NotificationType = "webhook"
)

// NotificationStatus определяет статус уведомления
type NotificationStatus string

const (
	NotificationStatusPending   NotificationStatus = "pending"
	NotificationStatusSent      NotificationStatus = "sent"
	NotificationStatusFailed    NotificationStatus = "failed"
	NotificationStatusDelivered NotificationStatus = "delivered"
)

// NotificationMessage представляет сообщение уведомления
type NotificationMessage struct {
	ID          string             `json:"id"`
	Type        NotificationType   `json:"type"`
	Status      NotificationStatus `json:"status"`
	Recipient   string            `json:"recipient"`
	Subject     string            `json:"subject"`
	Content     string            `json:"content"`
	Metadata    map[string]string `json:"metadata,omitempty"`
	RetryCount  int               `json:"retry_count"`
	CreatedAt   time.Time         `json:"created_at"`
	UpdatedAt   time.Time         `json:"updated_at"`
	ScheduledAt *time.Time        `json:"scheduled_at,omitempty"`
	SentAt      *time.Time        `json:"sent_at,omitempty"`
}

// NotificationTemplate представляет шаблон уведомления
type NotificationTemplate struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Type        NotificationType       `json:"type"`
	Subject     string                 `json:"subject"`
	Content     string                 `json:"content"`
	Variables   map[string]string      `json:"variables,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	IsActive    bool                   `json:"is_active"`
	CreatedAt   time.Time             `json:"created_at"`
	UpdatedAt   time.Time             `json:"updated_at"`
}

// NotificationPreferences представляет настройки уведомлений пользователя
type NotificationPreferences struct {
	UserID    string           `json:"user_id"`
	Channels  []NotificationType `json:"channels"`
	Enabled   bool             `json:"enabled"`
	Schedule  *NotificationSchedule `json:"schedule,omitempty"`
	UpdatedAt time.Time        `json:"updated_at"`
}

// NotificationSchedule представляет расписание уведомлений
type NotificationSchedule struct {
	TimeZone string   `json:"timezone"`
	StartTime string   `json:"start_time"` // Format: "HH:MM"
	EndTime   string   `json:"end_time"`   // Format: "HH:MM"
	Days      []string `json:"days"`       // ["monday", "tuesday", etc.]
}
