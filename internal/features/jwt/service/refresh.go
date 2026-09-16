package service

import (
	"context"
	"errors"
	"time"

	core_errors "github.com/Aam-Shaegar/Rhythm/internal/core/errors"
	"github.com/Aam-Shaegar/Rhythm/internal/features/jwt/domain"
)

func (s *JwtServiceImpl) RefreshTokens(ctx context.Context, refreshToken string) (*domain.TokenPair, error) {
	refreshTokenHash := s.hashToken(refreshToken)

	storedToken, err := s.jwtRepo.GetRefreshTokenByHash(ctx, refreshTokenHash)
	if err != nil {
		if errors.Is(err, core_errors.ErrNotFound) {
			return nil, core_errors.ErrUnauthorized
		}
		return nil, err
	}

	if storedToken.RevokedAt != nil {
		return nil, core_errors.ErrUnauthorized
	}

	if time.Now().UTC().After(storedToken.ExpiresAt) {
		return nil, core_errors.ErrUnauthorized
	}

	user, err := s.usersRepo.GetByID(ctx, storedToken.UserID)
	if err != nil {
		return nil, err
	}

	// Revoke old refresh token (rotation)
	if err := s.jwtRepo.RevokeRefreshToken(ctx, storedToken.ID); err != nil {
		return nil, err
	}

	return s.GeneratePair(ctx, user.ID, user.Username)
}

func (s *JwtServiceImpl) RevokeRefreshToken(ctx context.Context, tokenString string) error {
	refreshTokenHash := s.hashToken(tokenString)

	storedToken, err := s.jwtRepo.GetRefreshTokenByHash(ctx, refreshTokenHash)
	if err != nil {
		return err
	}

	return s.jwtRepo.RevokeRefreshToken(ctx, storedToken.ID)
}