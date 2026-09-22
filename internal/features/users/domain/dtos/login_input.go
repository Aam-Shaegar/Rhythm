package dtos

// Email принимает username или e-mail, JSON-имя оставлено для совместимости.
type LoginInput struct {
	Email    string `json:"email" validate:"required,min=3,max=255"`
	Password string `json:"password" validate:"required"`
}
