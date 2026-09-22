package domain

import (
	"time"

	"github.com/google/uuid"
)

type Reminder struct {
	ID         uuid.UUID
	UserID     uuid.UUID
	EntityType string
	EntityID   uuid.UUID
	Title      string
	RemindAt   time.Time
	IsSent     bool
	SentAt     *time.Time
	CreatedAt  time.Time
}

type ReminderSettings struct {
	EventReminders bool `json:"event_reminders"`
	TaskReminders  bool `json:"task_reminders"`
	EventBefore15m bool `json:"event_before_15m"`
	EventBefore1h  bool `json:"event_before_1h"`
	EventBefore24h bool `json:"event_before_24h"`
	TaskBefore1h   bool `json:"task_before_1h"`
	TaskBefore24h  bool `json:"task_before_24h"`
}

type UpdateReminderSettingsInput struct {
	EventReminders *bool `json:"event_reminders,omitempty"`
	TaskReminders  *bool `json:"task_reminders,omitempty"`
	EventBefore15m *bool `json:"event_before_15m,omitempty"`
	EventBefore1h  *bool `json:"event_before_1h,omitempty"`
	EventBefore24h *bool `json:"event_before_24h,omitempty"`
	TaskBefore1h   *bool `json:"task_before_1h,omitempty"`
	TaskBefore24h  *bool `json:"task_before_24h,omitempty"`
}

// Endpoint уникален: повторная подписка — upsert.
type PushSubscription struct {
	ID        uuid.UUID
	UserID    uuid.UUID
	Endpoint  string
	P256DH    string
	Auth      string
	CreatedAt time.Time
}

type PushSubscriptionInput struct {
	Endpoint string `json:"endpoint" validate:"required,url,max=2000"`
	P256DH   string `json:"p256dh" validate:"required,min=10,max=300"`
	Auth     string `json:"auth" validate:"required,min=10,max=300"`
}

type DeletePushSubscriptionInput struct {
	Endpoint string `json:"endpoint" validate:"required,url,max=2000"`
}
