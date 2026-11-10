package dtos

type UpdateProfileInput struct {
	CurrentPassword *string `json:"current_password,omitempty" validate:"omitempty,min=8,max=72"`
	NewPassword     *string `json:"new_password,omitempty" validate:"omitempty,min=8,max=72"`
	Username        *string `json:"username,omitempty" validate:"omitempty,min=3,max=100"`
	Email           *string `json:"email,omitempty" validate:"omitempty,email,max=255"`
}
