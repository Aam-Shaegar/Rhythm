package postgres

import (
	"context"

	"github.com/Aam-Shaegar/Rhythm/internal/features/jwt/domain"
	"github.com/google/uuid"
)

type JwtRepository interface {
	CreateRefreshToken(ctx context.Context, token *domain.RefreshToken) error
	GetRefreshTokenByHash(ctx context.Context, tokenHash string) (*domain.RefreshToken, error)
	RevokeRefreshToken(ctx context.Context, id uuid.UUID) error
	RevokeAllUserTokens(ctx context.Context, userID uuid.UUID) error
	DeleteExpiredTokens(ctx context.Context) (int64, error)
}
