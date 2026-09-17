package domain

type UpdateTaskInput struct {
	Title           *string         `json:"title,omitempty" validate:"omitempty,min=1,max=200"`
	Description     *string         `json:"description,omitempty" validate:"omitempty,max=2000"`
	DueAt           *string         `json:"due_at,omitempty" validate:"omitempty,datetime=2006-01-02T15:04:05Z07:00"`
	RecurrenceType  *RecurrenceType `json:"recurrence_type,omitempty" validate:"omitempty,oneof=daily weekly monthly yearly"`
	RecurrenceEnd   *string         `json:"recurrence_end,omitempty" validate:"omitempty,datetime=2006-01-02T15:04:05Z07:00"`
	// IsCompleted allows toggling completion both ways (F.8). Nil = no change.
	IsCompleted *bool `json:"is_completed,omitempty"`
}