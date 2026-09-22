package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"go.uber.org/zap"

	core_config "github.com/Aam-Shaegar/Rhythm/internal/core/config"
	core_errors "github.com/Aam-Shaegar/Rhythm/internal/core/errors"
	jwt_domain "github.com/Aam-Shaegar/Rhythm/internal/features/jwt/domain"
	users_domain "github.com/Aam-Shaegar/Rhythm/internal/features/users/domain"
	"github.com/google/uuid"
)

type mockJwtRepo struct {
	tokens map[string]*jwt_domain.RefreshToken
}

func newMockJwtRepo() *mockJwtRepo {
	return &mockJwtRepo{
		tokens: make(map[string]*jwt_domain.RefreshToken),
	}
}

func (m *mockJwtRepo) CreateRefreshToken(ctx context.Context, token *jwt_domain.RefreshToken) error {
	m.tokens[token.TokenHash] = token
	return nil
}

func (m *mockJwtRepo) GetRefreshTokenByHash(ctx context.Context, tokenHash string) (*jwt_domain.RefreshToken, error) {
	if t, ok := m.tokens[tokenHash]; ok {
		return t, nil
	}
	return nil, core_errors.ErrNotFound
}

func (m *mockJwtRepo) RevokeRefreshToken(ctx context.Context, id uuid.UUID) error {
	for _, t := range m.tokens {
		if t.ID == id {
			now := time.Now().UTC()
			t.RevokedAt = &now
			return nil
		}
	}
	return core_errors.ErrNotFound
}

func (m *mockJwtRepo) RevokeAllUserTokens(ctx context.Context, userID uuid.UUID) error {
	for _, t := range m.tokens {
		if t.UserID == userID && t.RevokedAt == nil {
			now := time.Now().UTC()
			t.RevokedAt = &now
		}
	}
	return nil
}

func (m *mockJwtRepo) DeleteExpiredTokens(ctx context.Context) (int64, error) {
	var count int64
	now := time.Now().UTC()
	for hash, t := range m.tokens {
		if t.ExpiresAt.Before(now) {
			delete(m.tokens, hash)
			count++
		}
	}
	return count, nil
}

type mockUsersRepo struct {
	users map[uuid.UUID]*users_domain.User
}

func newMockUsersRepo() *mockUsersRepo {
	return &mockUsersRepo{
		users: make(map[uuid.UUID]*users_domain.User),
	}
}

func (m *mockUsersRepo) GetByID(ctx context.Context, id uuid.UUID) (*users_domain.User, error) {
	if u, ok := m.users[id]; ok {
		return u, nil
	}
	return nil, core_errors.ErrNotFound
}

type mockLogger struct {
	msgs []string
}

func (m *mockLogger) Debug(msg string, fields ...zap.Field) {
	m.msgs = append(m.msgs, "DEBUG: "+msg)
}

func (m *mockLogger) Info(msg string, fields ...zap.Field) {
	m.msgs = append(m.msgs, "INFO: "+msg)
}

func (m *mockLogger) Warn(msg string, fields ...zap.Field) {
	m.msgs = append(m.msgs, "WARN: "+msg)
}

func (m *mockLogger) Error(msg string, fields ...zap.Field) {
	m.msgs = append(m.msgs, "ERROR: "+msg)
}

func TestGeneratePair_Success(t *testing.T) {
	jwtRepo := newMockJwtRepo()
	usersRepo := newMockUsersRepo()
	userID := uuid.New()
	usersRepo.users[userID] = &users_domain.User{
		ID:       userID,
		Username: "testuser",
		Email:    "test@example.com",
	}

	svc := NewJwtService(jwtRepo, usersRepo, &core_config.Config{
		JwtAccessSecret:  "test-secret",
		JwtRefreshSecret: "test-refresh-secret",
		JwtAccessTTL:     15 * time.Minute,
		JwtRefreshTTL:    24 * time.Hour,
	})

	pair, err := svc.GeneratePair(context.Background(), userID, "testuser")
	if err != nil {
		t.Fatalf("GeneratePair failed: %v", err)
	}

	if pair.AccessToken == "" {
		t.Error("expected access token")
	}
	if pair.RefreshToken == "" {
		t.Error("expected refresh token")
	}
	if pair.AccessTTL != 15*time.Minute {
		t.Errorf("expected access TTL 15m, got %v", pair.AccessTTL)
	}
}

func TestValidateAccessToken_Valid(t *testing.T) {
	jwtRepo := newMockJwtRepo()
	usersRepo := newMockUsersRepo()
	userID := uuid.New()
	usersRepo.users[userID] = &users_domain.User{
		ID:       userID,
		Username: "testuser",
		Email:    "test@example.com",
	}

	svc := NewJwtService(jwtRepo, usersRepo, &core_config.Config{
		JwtAccessSecret:  "test-secret",
		JwtRefreshSecret: "test-refresh-secret",
		JwtAccessTTL:     15 * time.Minute,
		JwtRefreshTTL:    24 * time.Hour,
	})

	pair, err := svc.GeneratePair(context.Background(), userID, "testuser")
	if err != nil {
		t.Fatalf("GeneratePair failed: %v", err)
	}

	userIDStr, username, err := svc.ValidateAccessToken(pair.AccessToken)
	if err != nil {
		t.Fatalf("ValidateAccessToken failed: %v", err)
	}

	if userIDStr != userID.String() {
		t.Errorf("expected userID %s, got %s", userID.String(), userIDStr)
	}
	if username != "testuser" {
		t.Errorf("expected username testuser, got %s", username)
	}
}

func TestValidateAccessToken_Invalid(t *testing.T) {
	jwtRepo := newMockJwtRepo()
	usersRepo := newMockUsersRepo()

	svc := NewJwtService(jwtRepo, usersRepo, &core_config.Config{
		JwtAccessSecret:  "test-secret",
		JwtRefreshSecret: "test-refresh-secret",
		JwtAccessTTL:     15 * time.Minute,
		JwtRefreshTTL:    24 * time.Hour,
	})

	_, _, err := svc.ValidateAccessToken("invalid-token")
	if err == nil {
		t.Fatal("expected error for invalid token")
	}
	if !errors.Is(err, core_errors.ErrUnauthorized) {
		t.Errorf("expected ErrUnauthorized, got %v", err)
	}
}

func TestRefreshTokens_Success(t *testing.T) {
	jwtRepo := newMockJwtRepo()
	usersRepo := newMockUsersRepo()
	userID := uuid.New()
	usersRepo.users[userID] = &users_domain.User{
		ID:       userID,
		Username: "testuser",
		Email:    "test@example.com",
	}

	svc := NewJwtService(jwtRepo, usersRepo, &core_config.Config{
		JwtAccessSecret:  "test-secret",
		JwtRefreshSecret: "test-refresh-secret",
		JwtAccessTTL:     15 * time.Minute,
		JwtRefreshTTL:    24 * time.Hour,
	})

	pair, err := svc.GeneratePair(context.Background(), userID, "testuser")
	if err != nil {
		t.Fatalf("GeneratePair failed: %v", err)
	}

	newPair, err := svc.RefreshTokens(context.Background(), pair.RefreshToken)
	if err != nil {
		t.Fatalf("RefreshTokens failed: %v", err)
	}

	if newPair.AccessToken == "" {
		t.Error("expected new access token")
	}
	if newPair.RefreshToken == "" {
		t.Error("expected new refresh token")
	}
	if newPair.RefreshToken == pair.RefreshToken {
		t.Error("expected new refresh token to be different")
	}
}

func TestRefreshTokens_Revoked(t *testing.T) {
	jwtRepo := newMockJwtRepo()
	usersRepo := newMockUsersRepo()
	userID := uuid.New()
	usersRepo.users[userID] = &users_domain.User{
		ID:       userID,
		Username: "testuser",
		Email:    "test@example.com",
	}

	svc := NewJwtService(jwtRepo, usersRepo, &core_config.Config{
		JwtAccessSecret:  "test-secret",
		JwtRefreshSecret: "test-refresh-secret",
		JwtAccessTTL:     15 * time.Minute,
		JwtRefreshTTL:    24 * time.Hour,
	})

	pair, err := svc.GeneratePair(context.Background(), userID, "testuser")
	if err != nil {
		t.Fatalf("GeneratePair failed: %v", err)
	}

	for _, t := range jwtRepo.tokens {
		if t.UserID == userID {
			now := time.Now().UTC()
			t.RevokedAt = &now
		}
	}

	_, err = svc.RefreshTokens(context.Background(), pair.RefreshToken)
	if err == nil {
		t.Fatal("expected error for revoked token")
	}
	if !errors.Is(err, core_errors.ErrUnauthorized) {
		t.Errorf("expected ErrUnauthorized, got %v", err)
	}
}

func TestStartCleanup(t *testing.T) {
	jwtRepo := newMockJwtRepo()
	usersRepo := newMockUsersRepo()

	svc := NewJwtService(jwtRepo, usersRepo, &core_config.Config{
		JwtAccessSecret:  "test-secret",
		JwtRefreshSecret: "test-refresh-secret",
		JwtAccessTTL:     15 * time.Minute,
		JwtRefreshTTL:    24 * time.Hour,
	})

	logger := &mockLogger{}
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	svc.StartCleanup(ctx, 10*time.Millisecond, logger)

	time.Sleep(150 * time.Millisecond)

	found := false
	for _, msg := range logger.msgs {
		if len(msg) > 0 {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected cleanup to log something")
	}
}
