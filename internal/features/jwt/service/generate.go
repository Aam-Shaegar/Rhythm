package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"time"

	core_config "github.com/Aam-Shaegar/Rhythm/internal/core/config"
	"github.com/Aam-Shaegar/Rhythm/internal/features/jwt/domain"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type JwtServiceImpl struct {
	jwtRepo    JwtRepository
	usersRepo  UsersRepository
	cfg        *core_config.Config
	accessTTL  time.Duration
	refreshTTL time.Duration
}

func NewJwtService(jwtRepo JwtRepository, usersRepo UsersRepository, cfg *core_config.Config) *JwtServiceImpl {
	return &JwtServiceImpl{
		jwtRepo:    jwtRepo,
		usersRepo:  usersRepo,
		cfg:        cfg,
		accessTTL:  cfg.JwtAccessTTL,
		refreshTTL: cfg.JwtRefreshTTL,
	}
}

func (s *JwtServiceImpl) GeneratePair(ctx context.Context, userID uuid.UUID, username string) (*domain.TokenPair, error) {
	accessToken, err := s.generateAccessToken(userID, username)
	if err != nil {
		return nil, err
	}

	refreshToken, err := s.generateRefreshToken()
	if err != nil {
		return nil, err
	}

	refreshTokenHash := s.hashToken(refreshToken)
	expiresAt := time.Now().UTC().Add(s.refreshTTL)

	rt := &domain.RefreshToken{
		ID:        uuid.New(),
		UserID:    userID,
		TokenHash: refreshTokenHash,
		ExpiresAt: expiresAt,
		CreatedAt: time.Now().UTC(),
	}

	if err := s.jwtRepo.CreateRefreshToken(ctx, rt); err != nil {
		return nil, err
	}

	return &domain.TokenPair{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		AccessTTL:    s.accessTTL,
		RefreshTTL:   s.refreshTTL,
	}, nil
}

func (s *JwtServiceImpl) generateAccessToken(userID uuid.UUID, username string) (string, error) {
	claims := jwt.MapClaims{
		"uid":   userID.String(),
		"uname": username,
		"exp":   time.Now().UTC().Add(s.accessTTL).Unix(),
		"iat":   time.Now().UTC().Unix(),
		"type":  "access",
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(s.cfg.JwtAccessSecret))
}

func (s *JwtServiceImpl) generateRefreshToken() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

func (s *JwtServiceImpl) hashToken(token string) string {
	hash := sha256.Sum256([]byte(token))
	return hex.EncodeToString(hash[:])
}
