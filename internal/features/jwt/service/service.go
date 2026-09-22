package service

import (
	"context"
	"time"

	"go.uber.org/zap"

	jwt_domain "github.com/Aam-Shaegar/Rhythm/internal/features/jwt/domain"
	users_domain "github.com/Aam-Shaegar/Rhythm/internal/features/users/domain"
	"github.com/google/uuid"
)

type JwtRepository interface {
	CreateRefreshToken(ctx context.Context, token *jwt_domain.RefreshToken) error
	GetRefreshTokenByHash(ctx context.Context, tokenHash string) (*jwt_domain.RefreshToken, error)
	RevokeRefreshToken(ctx context.Context, id uuid.UUID) error
	RevokeAllUserTokens(ctx context.Context, userID uuid.UUID) error
	DeleteExpiredTokens(ctx context.Context) (int64, error)
}

type UsersRepository interface {
	GetByID(ctx context.Context, id uuid.UUID) (*users_domain.User, error)
}

type JwtService interface {
	GeneratePair(ctx context.Context, userID uuid.UUID, username string) (*jwt_domain.TokenPair, error)
	ValidateAccessToken(tokenString string) (string, string, error)
	RefreshTokens(ctx context.Context, refreshToken string) (*jwt_domain.TokenPair, error)
	RevokeRefreshToken(ctx context.Context, tokenString string) error
	StartCleanup(ctx context.Context, interval time.Duration, logger Logger)
}

type Logger interface {
	Debug(msg string, fields ...zap.Field)
	Info(msg string, fields ...zap.Field)
	Warn(msg string, fields ...zap.Field)
	Error(msg string, fields ...zap.Field)
}
