package domain

type CreateEventInput struct {
	Title       string  `json:"title" validate:"required,min=1,max=200"`
	Description *string `json:"description,omitempty" validate:"omitempty,max=2000"`
	StartAt     string  `json:"start_at" validate:"required,datetime=2006-01-02T15:04:05Z07:00"`
	EndAt       string  `json:"end_at" validate:"required,datetime=2006-01-02T15:04:05Z07:00"`
}
