package service

import (
	"context"
	"errors"
	"github.com/google/uuid"
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

	pair, next, err := s.newPair(user.ID, user.Username)
	if err != nil {
		return nil, err
	}
	if err := s.jwtRepo.RotateRefreshToken(ctx, storedToken.ID, next); err != nil {
		return nil, err
	}
	return pair, nil
}

func (s *JwtServiceImpl) RevokeRefreshToken(ctx context.Context, tokenString string) error {
	refreshTokenHash := s.hashToken(tokenString)

	storedToken, err := s.jwtRepo.GetRefreshTokenByHash(ctx, refreshTokenHash)
	if err != nil {
		return err
	}

	return s.jwtRepo.RevokeRefreshToken(ctx, storedToken.ID)
}

func (s *JwtServiceImpl) RevokeAllUserSessions(ctx context.Context, uid uuid.UUID) error {
	return s.jwtRepo.RevokeAllUserTokens(ctx, uid)
}
