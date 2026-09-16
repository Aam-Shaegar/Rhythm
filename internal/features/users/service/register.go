package service

import (
	"context"
	"errors"
	"time"

	core_errors "github.com/Aam-Shaegar/Rhythm/internal/core/errors"
	"github.com/Aam-Shaegar/Rhythm/internal/features/jwt/service"
	"github.com/Aam-Shaegar/Rhythm/internal/features/users/domain"
	"github.com/Aam-Shaegar/Rhythm/internal/features/users/domain/dtos"
	"github.com/Aam-Shaegar/Rhythm/internal/features/users/repository/postgres"
	"golang.org/x/crypto/bcrypt"
	"github.com/google/uuid"
)

type UsersServiceImpl struct {
	repo     postgres.UsersRepository
	jwtSvc   service.JwtService
	passwordHasher PasswordHasher
}

type PasswordHasher interface {
	Hash(password string) (string, error)
	Compare(hash, password string) error
}

type BcryptHasher struct{}

func (BcryptHasher) Hash(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(bytes), err
}

func (BcryptHasher) Compare(hash, password string) error {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
}

func NewUsersService(repo postgres.UsersRepository, jwtSvc service.JwtService) *UsersServiceImpl {
	return &UsersServiceImpl{
		repo:           repo,
		jwtSvc:         jwtSvc,
		passwordHasher: BcryptHasher{},
	}
}

func (s *UsersServiceImpl) Register(ctx context.Context, input dtos.RegisterInput) (*dtos.AuthResponse, error) {
	exists, err := s.repo.ExistsByEmail(ctx, input.Email)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, core_errors.ErrConflict
	}

	exists, err = s.repo.ExistsByUsername(ctx, input.Username)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, core_errors.ErrConflict
	}

	passwordHash, err := s.passwordHasher.Hash(input.Password)
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	user := &domain.User{
		ID:           uuid.New(),
		Username:     input.Username,
		Email:        input.Email,
		PasswordHash: passwordHash,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	if err := s.repo.Create(ctx, user); err != nil {
		return nil, err
	}

	return s.issueAuthResponse(ctx, user)
}

func (s *UsersServiceImpl) Login(ctx context.Context, input dtos.LoginInput) (*dtos.AuthResponse, error) {
	user, err := s.repo.GetByEmail(ctx, input.Email)
	if err != nil {
		if errors.Is(err, core_errors.ErrNotFound) {
			return nil, core_errors.ErrUnauthorized
		}
		return nil, err
	}

	if err := s.passwordHasher.Compare(user.PasswordHash, input.Password); err != nil {
		return nil, core_errors.ErrUnauthorized
	}

	return s.issueAuthResponse(ctx, user)
}

func (s *UsersServiceImpl) GetMe(ctx context.Context, userID uuid.UUID) (*dtos.UserResponse, error) {
	user, err := s.repo.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	return &dtos.UserResponse{
		ID:        user.ID,
		Username:  user.Username,
		Email:     user.Email,
		CreatedAt: user.CreatedAt,
		UpdatedAt: user.UpdatedAt,
	}, nil
}

func (s *UsersServiceImpl) UpdateProfile(ctx context.Context, userID uuid.UUID, input dtos.UpdateProfileInput) (*dtos.UserResponse, error) {
	user, err := s.repo.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	if input.Username != nil {
		exists, err := s.repo.ExistsByUsername(ctx, *input.Username)
		if err != nil {
			return nil, err
		}
		if exists && *input.Username != user.Username {
			return nil, core_errors.ErrConflict
		}
		user.Username = *input.Username
	}

	if input.Email != nil {
		exists, err := s.repo.ExistsByEmail(ctx, *input.Email)
		if err != nil {
			return nil, err
		}
		if exists && *input.Email != user.Email {
			return nil, core_errors.ErrConflict
		}
		user.Email = *input.Email
	}

	user.UpdatedAt = time.Now().UTC()

	if err := s.repo.Update(ctx, user); err != nil {
		return nil, err
	}

	return &dtos.UserResponse{
		ID:        user.ID,
		Username:  user.Username,
		Email:     user.Email,
		CreatedAt: user.CreatedAt,
		UpdatedAt: user.UpdatedAt,
	}, nil
}

func (s *UsersServiceImpl) issueAuthResponse(ctx context.Context, user *domain.User) (*dtos.AuthResponse, error) {
	tokenPair, err := s.jwtSvc.GeneratePair(ctx, user.ID, user.Username)
	if err != nil {
		return nil, err
	}

	return &dtos.AuthResponse{
		User: dtos.UserResponse{
			ID:        user.ID,
			Username:  user.Username,
			Email:     user.Email,
			CreatedAt: user.CreatedAt,
			UpdatedAt: user.UpdatedAt,
		},
		AccessToken:  tokenPair.AccessToken,
		RefreshToken: tokenPair.RefreshToken,
	}, nil
}