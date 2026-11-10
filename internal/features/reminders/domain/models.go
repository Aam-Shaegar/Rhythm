package domain

import (
	"fmt"
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
	IsOverdue  bool
	IsSent     bool
	SentAt     *time.Time
	CreatedAt  time.Time
}

// Offsets are minutes before the start/deadline; zero means at the deadline.
type ReminderSettings struct {
	EventOffsets []int `json:"event_offsets"`
	TaskOffsets  []int `json:"task_offsets"`
}

type UpdateReminderSettingsInput struct {
	EventOffsets *[]int `json:"event_offsets,omitempty"`
	TaskOffsets  *[]int `json:"task_offsets,omitempty"`
}

func DefaultSettings() ReminderSettings {
	return ReminderSettings{EventOffsets: []int{60, 0}, TaskOffsets: []int{60, 0}}
}

func (s UpdateReminderSettingsInput) Validate() error {
	for _, offsets := range []*[]int{s.EventOffsets, s.TaskOffsets} {
		if offsets == nil {
			continue
		}
		if len(*offsets) > 5 {
			return fmt.Errorf("maximum five reminders per category")
		}
		seen := map[int]bool{}
		for _, n := range *offsets {
			if n < 0 || n > 525600 || seen[n] {
				return fmt.Errorf("offset must be unique and between 0 and 525600 minutes")
			}
			seen[n] = true
		}
	}
	return nil
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
