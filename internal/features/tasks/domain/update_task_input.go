package domain

type UpdateTaskInput struct {
	Importance       *int            `json:"importance,omitempty" validate:"omitempty,min=1,max=3"`
	EstimatedMinutes *int            `json:"estimated_minutes,omitempty" validate:"omitempty,min=5,max=480"`
	Title            *string         `json:"title,omitempty" validate:"omitempty,min=1,max=200"`
	Description      *string         `json:"description,omitempty" validate:"omitempty,max=2000"`
	DueAt            *string         `json:"due_at,omitempty" validate:"omitempty,datetime=2006-01-02T15:04:05Z07:00"`
	RecurrenceType   *RecurrenceType `json:"recurrence_type,omitempty" validate:"omitempty,oneof=daily weekly monthly yearly"`
	RecurrenceEnd    *string         `json:"recurrence_end,omitempty" validate:"omitempty,datetime=2006-01-02T15:04:05Z07:00"`
	// Nil = без изменений.
	IsCompleted *bool `json:"is_completed,omitempty"`
}
