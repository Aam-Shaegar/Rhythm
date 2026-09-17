package dtos

// LoginInput accepts either a username or an e-mail address in Email
// (TZ U.8/F.2: «логин или электронная почта»). The JSON field name stays
// "email" for API compatibility.
type LoginInput struct {
	Email    string `json:"email" validate:"required,min=3,max=255"`
	Password string `json:"password" validate:"required"`
}