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
	RemindAt   time.Time
	IsSent     bool
	SentAt     *time.Time
	CreatedAt  time.Time
}

type CreateReminderInput struct {
	UserID     uuid.UUID
	EntityType string
	EntityID   uuid.UUID
	RemindAt   time.Time
}

type ReminderSettings struct {
	EventReminders  bool `json:"event_reminders"`
	TaskReminders   bool `json:"task_reminders"`
	EventBefore15m  bool `json:"event_before_15m"`
	EventBefore1h   bool `json:"event_before_1h"`
	EventBefore24h  bool `json:"event_before_24h"`
	TaskBefore1h    bool `json:"task_before_1h"`
	TaskBefore24h   bool `json:"task_before_24h"`
}

type UpdateReminderSettingsInput struct {
	EventReminders  *bool `json:"event_reminders,omitempty"`
	TaskReminders   *bool `json:"task_reminders,omitempty"`
	EventBefore15m  *bool `json:"event_before_15m,omitempty"`
	EventBefore1h   *bool `json:"event_before_1h,omitempty"`
	EventBefore24h  *bool `json:"event_before_24h,omitempty"`
	TaskBefore1h    *bool `json:"task_before_1h,omitempty"`
	TaskBefore24h   *bool `json:"task_before_24h,omitempty"`
}

// Simplified types for cross-feature references
type Event struct {
	ID          uuid.UUID
	UserID      uuid.UUID
	Title       string
	Description *string
	StartAt     time.Time
	EndAt       time.Time
}

type Task struct {
	ID              uuid.UUID
	UserID          uuid.UUID
	Title           string
	Description     *string
	DueAt           time.Time
	IsCompleted     bool
	RecurrenceType  *string
	RecurrenceEnd   *time.Time
	ParentTaskID    *uuid.UUID
}