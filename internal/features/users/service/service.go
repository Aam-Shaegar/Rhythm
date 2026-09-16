package service

import (
	"context"

	"github.com/Aam-Shaegar/Rhythm/internal/features/users/domain/dtos"
	"github.com/google/uuid"
)

type UsersService interface {
	Register(ctx context.Context, input dtos.RegisterInput) (*dtos.AuthResponse, error)
	Login(ctx context.Context, input dtos.LoginInput) (*dtos.AuthResponse, error)
	GetMe(ctx context.Context, userID uuid.UUID) (*dtos.UserResponse, error)
	UpdateProfile(ctx context.Context, userID uuid.UUID, input dtos.UpdateProfileInput) (*dtos.UserResponse, error)
}