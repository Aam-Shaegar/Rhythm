package service

import (
	"context"
	"errors"
	"testing"
	"time"

	core_errors "github.com/Aam-Shaegar/Rhythm/internal/core/errors"
	jwt_domain "github.com/Aam-Shaegar/Rhythm/internal/features/jwt/domain"
	"github.com/Aam-Shaegar/Rhythm/internal/features/jwt/service"
	users_domain "github.com/Aam-Shaegar/Rhythm/internal/features/users/domain"
	"github.com/Aam-Shaegar/Rhythm/internal/features/users/domain/dtos"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

type mockUsersRepo struct {
	users    map[string]*users_domain.User
	emails   map[string]*users_domain.User
	byID     map[uuid.UUID]*users_domain.User
}

func newMockUsersRepo() *mockUsersRepo {
	return &mockUsersRepo{
		users:  make(map[string]*users_domain.User),
		emails: make(map[string]*users_domain.User),
		byID:   make(map[uuid.UUID]*users_domain.User),
	}
}

func (m *mockUsersRepo) Create(ctx context.Context, user *users_domain.User) error {
	if _, exists := m.emails[user.Email]; exists {
		return core_errors.ErrConflict
	}
	if _, exists := m.users[user.Username]; exists {
		return core_errors.ErrConflict
	}
	m.users[user.Username] = user
	m.emails[user.Email] = user
	m.byID[user.ID] = user
	return nil
}

func (m *mockUsersRepo) GetByID(ctx context.Context, id uuid.UUID) (*users_domain.User, error) {
	if u, ok := m.byID[id]; ok {
		return u, nil
	}
	return nil, core_errors.ErrNotFound
}

func (m *mockUsersRepo) GetByEmail(ctx context.Context, email string) (*users_domain.User, error) {
	if u, ok := m.emails[email]; ok {
		return u, nil
	}
	return nil, core_errors.ErrNotFound
}

func (m *mockUsersRepo) GetByUsername(ctx context.Context, username string) (*users_domain.User, error) {
	if u, ok := m.users[username]; ok {
		return u, nil
	}
	return nil, core_errors.ErrNotFound
}

func (m *mockUsersRepo) ExistsByEmail(ctx context.Context, email string) (bool, error) {
	_, ok := m.emails[email]
	return ok, nil
}

func (m *mockUsersRepo) ExistsByUsername(ctx context.Context, username string) (bool, error) {
	_, ok := m.users[username]
	return ok, nil
}

func (m *mockUsersRepo) Update(ctx context.Context, user *users_domain.User) error {
	if _, ok := m.byID[user.ID]; !ok {
		return core_errors.ErrNotFound
	}
	// Remove old username/email from maps if changed
	delete(m.users, m.byID[user.ID].Username)
	delete(m.emails, m.byID[user.ID].Email)
	m.users[user.Username] = user
	m.emails[user.Email] = user
	m.byID[user.ID] = user
	return nil
}

type mockJwtService struct {
	tokenPair *jwt_domain.TokenPair
	err error
}

func newMockJwtService() *mockJwtService {
	return &mockJwtService{
		tokenPair: &jwt_domain.TokenPair{
			AccessToken:  "test-access-token",
			RefreshToken: "test-refresh-token",
			AccessTTL:    15 * 60,
			RefreshTTL:   720 * 60 * 60,
		},
	}
}

func (m *mockJwtService) GeneratePair(ctx context.Context, userID uuid.UUID, username string) (*jwt_domain.TokenPair, error) {
	return m.tokenPair, m.err
}

func (m *mockJwtService) ValidateAccessToken(tokenString string) (string, string, error) {
	return "test-user-id", "test-username", m.err
}

func (m *mockJwtService) RefreshTokens(ctx context.Context, refreshToken string) (*jwt_domain.TokenPair, error) {
	return m.tokenPair, m.err
}

func (m *mockJwtService) RevokeRefreshToken(ctx context.Context, tokenString string) error {
	return m.err
}

func (m *mockJwtService) StartCleanup(ctx context.Context, interval time.Duration, logger service.Logger) {}

func TestRegister_Success(t *testing.T) {
	repo := newMockUsersRepo()
	jwtSvc := newMockJwtService()
	svc := NewUsersService(repo, jwtSvc)

	input := dtos.RegisterInput{
		Username: "testuser",
		Email:    "test@example.com",
		Password: "password123",
	}

	resp, err := svc.Register(context.Background(), input)
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	if resp.User.Username != "testuser" {
		t.Errorf("expected username testuser, got %s", resp.User.Username)
	}
	if resp.User.Email != "test@example.com" {
		t.Errorf("expected email test@example.com, got %s", resp.User.Email)
	}
	if resp.AccessToken != "test-access-token" {
		t.Errorf("expected access token, got %s", resp.AccessToken)
	}
}

func TestRegister_DuplicateEmail(t *testing.T) {
	repo := newMockUsersRepo()
	jwtSvc := newMockJwtService()
	svc := NewUsersService(repo, jwtSvc)

	// First user
	hash, _ := bcrypt.GenerateFromPassword([]byte("pass"), bcrypt.DefaultCost)
	repo.Create(context.Background(), &users_domain.User{
		ID:           uuid.New(),
		Username:     "user1",
		Email:        "test@example.com",
		PasswordHash: string(hash),
	})

	// Try to register with same email
	input := dtos.RegisterInput{
		Username: "user2",
		Email:    "test@example.com",
		Password: "password123",
	}

	_, err := svc.Register(context.Background(), input)
	if err == nil {
		t.Fatal("expected conflict error for duplicate email")
	}
	if !errors.Is(err, core_errors.ErrConflict) {
		t.Errorf("expected ErrConflict, got %v", err)
	}
}

func TestLogin_Success(t *testing.T) {
	repo := newMockUsersRepo()
	jwtSvc := newMockJwtService()
	svc := NewUsersService(repo, jwtSvc)

	hash, _ := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)
	user := &users_domain.User{
		ID:           uuid.New(),
		Username:     "testuser",
		Email:        "test@example.com",
		PasswordHash: string(hash),
	}
	repo.Create(context.Background(), user)

	input := dtos.LoginInput{
		Email:    "test@example.com",
		Password: "password123",
	}

	resp, err := svc.Login(context.Background(), input)
	if err != nil {
		t.Fatalf("Login failed: %v", err)
	}

	if resp.User.Username != "testuser" {
		t.Errorf("expected username testuser, got %s", resp.User.Username)
	}
}

func TestLogin_WrongPassword(t *testing.T) {
	repo := newMockUsersRepo()
	jwtSvc := newMockJwtService()
	svc := NewUsersService(repo, jwtSvc)

	hash, _ := bcrypt.GenerateFromPassword([]byte("password123"), bcrypt.DefaultCost)
	user := &users_domain.User{
		ID:           uuid.New(),
		Username:     "testuser",
		Email:        "test@example.com",
		PasswordHash: string(hash),
	}
	repo.Create(context.Background(), user)

	input := dtos.LoginInput{
		Email:    "test@example.com",
		Password: "wrongpassword",
	}

	_, err := svc.Login(context.Background(), input)
	if err == nil {
		t.Fatal("expected unauthorized error for wrong password")
	}
	if !errors.Is(err, core_errors.ErrUnauthorized) {
		t.Errorf("expected ErrUnauthorized, got %v", err)
	}
}

func TestGetMe_Success(t *testing.T) {
	repo := newMockUsersRepo()
	jwtSvc := newMockJwtService()
	svc := NewUsersService(repo, jwtSvc)

	userID := uuid.New()
	user := &users_domain.User{
		ID:           userID,
		Username:     "testuser",
		Email:        "test@example.com",
		PasswordHash: "hash",
	}
	repo.Create(context.Background(), user)

	resp, err := svc.GetMe(context.Background(), userID)
	if err != nil {
		t.Fatalf("GetMe failed: %v", err)
	}

	if resp.ID != userID {
		t.Errorf("expected user ID %v, got %v", userID, resp.ID)
	}
}

func TestGetMe_NotFound(t *testing.T) {
	repo := newMockUsersRepo()
	jwtSvc := newMockJwtService()
	svc := NewUsersService(repo, jwtSvc)

	_, err := svc.GetMe(context.Background(), uuid.New())
	if err == nil {
		t.Fatal("expected not found error")
	}
	if !errors.Is(err, core_errors.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestUpdateProfile_Success(t *testing.T) {
	repo := newMockUsersRepo()
	jwtSvc := newMockJwtService()
	svc := NewUsersService(repo, jwtSvc)

	userID := uuid.New()
	user := &users_domain.User{
		ID:           userID,
		Username:     "oldname",
		Email:        "old@example.com",
		PasswordHash: "hash",
	}
	repo.Create(context.Background(), user)

	input := dtos.UpdateProfileInput{
		Username: strPtr("newname"),
		Email:    strPtr("new@example.com"),
	}

	resp, err := svc.UpdateProfile(context.Background(), userID, input)
	if err != nil {
		t.Fatalf("UpdateProfile failed: %v", err)
	}

	if resp.Username != "newname" {
		t.Errorf("expected username newname, got %s", resp.Username)
	}
	if resp.Email != "new@example.com" {
		t.Errorf("expected email new@example.com, got %s", resp.Email)
	}
}

func strPtr(s string) *string {
	return &s
}