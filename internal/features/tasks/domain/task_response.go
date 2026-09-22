package domain

import (
	"time"

	"github.com/google/uuid"
)

type TaskResponse struct {
	ID             uuid.UUID       `json:"id"`
	Title          string          `json:"title"`
	Description    *string         `json:"description,omitempty"`
	DueAt          time.Time       `json:"due_at"`
	IsCompleted    bool            `json:"is_completed"`
	CompletedAt    *time.Time      `json:"completed_at,omitempty"`
	RecurrenceType *RecurrenceType `json:"recurrence_type,omitempty"`
	RecurrenceEnd  *time.Time      `json:"recurrence_end,omitempty"`
	ParentTaskID   *uuid.UUID      `json:"parent_task_id,omitempty"`
	CreatedAt      time.Time       `json:"created_at"`
	UpdatedAt      time.Time       `json:"updated_at"`
}
