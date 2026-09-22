package domain

type UpdateEventInput struct {
	Title       *string `json:"title,omitempty" validate:"omitempty,min=1,max=200"`
	Description *string `json:"description,omitempty" validate:"omitempty,max=2000"`
	StartAt     *string `json:"start_at,omitempty" validate:"omitempty,datetime=2006-01-02T15:04:05Z07:00"`
	EndAt       *string `json:"end_at,omitempty" validate:"omitempty,datetime=2006-01-02T15:04:05Z07:00"`
}
