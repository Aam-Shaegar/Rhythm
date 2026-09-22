package dtos

type UpdateProfileInput struct {
	Username *string `json:"username,omitempty" validate:"omitempty,min=3,max=100"`
	Email    *string `json:"email,omitempty" validate:"omitempty,email,max=255"`
}
