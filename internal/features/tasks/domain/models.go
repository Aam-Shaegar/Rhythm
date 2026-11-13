package domain

import (
	"time"

	"github.com/google/uuid"
)

type Task struct {
	Importance       int
	EstimatedMinutes int
	ID               uuid.UUID
	UserID           uuid.UUID
	Title            string
	Description      *string
	DueAt            time.Time
	IsCompleted      bool
	CompletedAt      *time.Time
	RecurrenceType   *RecurrenceType
	RecurrenceEnd    *time.Time
	ParentTaskID     *uuid.UUID
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

// CompletedLate preserves the distinction between finishing a task and meeting its deadline.
func (t *Task) CompletedLate() bool {
	return t.IsCompleted && t.CompletedAt != nil && t.CompletedAt.After(t.DueAt)
}

func (t *Task) OverdueAt(now time.Time) bool {
	return t.CompletedLate() || (!t.IsCompleted && t.DueAt.Before(now))
}

type RecurrenceType string

const (
	RecurrenceDaily   RecurrenceType = "daily"
	RecurrenceWeekly  RecurrenceType = "weekly"
	RecurrenceMonthly RecurrenceType = "monthly"
	RecurrenceYearly  RecurrenceType = "yearly"
)

type TaskFilter struct {
	UserID    uuid.UUID  `query:"-"`
	DateFrom  *time.Time `query:"date_from"`
	DateTo    *time.Time `query:"date_to"`
	Completed *bool      `query:"completed"`
	Recurring *bool      `query:"recurring"`
	Limit     int        `query:"limit" validate:"omitempty,min=0,max=1000"`
	Offset    int        `query:"offset" validate:"omitempty,min=0"`
}
