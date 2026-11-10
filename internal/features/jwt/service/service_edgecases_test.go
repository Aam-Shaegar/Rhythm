package service

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	core_config "github.com/Aam-Shaegar/Rhythm/internal/core/config"
	core_errors "github.com/Aam-Shaegar/Rhythm/internal/core/errors"
	jwt_domain "github.com/Aam-Shaegar/Rhythm/internal/features/jwt/domain"
	users_domain "github.com/Aam-Shaegar/Rhythm/internal/features/users/domain"
	"github.com/google/uuid"
)

func newEdgeSvc() (*JwtServiceImpl, *mockJwtRepo, *mockUsersRepo) {
	jwtRepo := newMockJwtRepo()
	usersRepo := newMockUsersRepo()
	svc := NewJwtService(jwtRepo, usersRepo, &core_config.Config{
		JwtAccessSecret:  "test-secret",
		JwtRefreshSecret: "test-refresh-secret",
		JwtAccessTTL:     15 * time.Minute,
		JwtRefreshTTL:    24 * time.Hour,
	})
	return svc, jwtRepo, usersRepo
}

func edgeUser(uid uuid.UUID, name string) *users_domain.User {
	return &users_domain.User{ID: uid, Username: name, Email: name + "@ex.com"}
}

func TestValidate_MalformedTokens(t *testing.T) {
	svc, _, _ := newEdgeSvc()
	for _, bad := range []string{"", "abc", "a.b.c", "Bearer xxx", strings.Repeat("x", 500)} {
		if _, _, err := svc.ValidateAccessToken(bad); !errors.Is(err, core_errors.ErrUnauthorized) {
			t.Fatalf("token %q: expected Unauthorized, got %v", bad, err)
		}
	}
}

func TestValidate_WrongSecret(t *testing.T) {
	svc1, _, users1 := newEdgeSvc()
	uid := uuid.New()
	users1.users[uid] = edgeUser(uid, "bob")
	pair, err := svc1.GeneratePair(context.Background(), uid, "bob")
	if err != nil {
		t.Fatal(err)
	}
	svc2 := NewJwtService(newMockJwtRepo(), newMockUsersRepo(), &core_config.Config{
		JwtAccessSecret:  "other-access",
		JwtRefreshSecret: "other-refresh",
		JwtAccessTTL:     15 * time.Minute,
		JwtRefreshTTL:    24 * time.Hour,
	})
	if _, _, err := svc2.ValidateAccessToken(pair.AccessToken); !errors.Is(err, core_errors.ErrUnauthorized) {
		t.Fatalf("wrong secret should be Unauthorized, got %v", err)
	}
}

func TestRefresh_UnknownToken(t *testing.T) {
	svc, _, _ := newEdgeSvc()
	if _, err := svc.RefreshTokens(context.Background(), "no-such-token"); !errors.Is(err, core_errors.ErrUnauthorized) {
		t.Fatalf("expected Unauthorized, got %v", err)
	}
}

func TestRefresh_ExpiredToken(t *testing.T) {
	svc, repo, users := newEdgeSvc()
	uid := uuid.New()
	users.users[uid] = edgeUser(uid, "bob")
	pair, err := svc.GeneratePair(context.Background(), uid, "bob")
	if err != nil {
		t.Fatal(err)
	}
	for _, tok := range repo.tokens {
		past := time.Now().UTC().Add(-time.Hour)
		tok.ExpiresAt = past
	}
	if _, err := svc.RefreshTokens(context.Background(), pair.RefreshToken); !errors.Is(err, core_errors.ErrUnauthorized) {
		t.Fatalf("expired should be Unauthorized, got %v", err)
	}
}

func TestRefresh_Rotation_RevokesOld(t *testing.T) {
	svc, _, users := newEdgeSvc()
	uid := uuid.New()
	users.users[uid] = edgeUser(uid, "bob")
	pair, _ := svc.GeneratePair(context.Background(), uid, "bob")
	newPair, err := svc.RefreshTokens(context.Background(), pair.RefreshToken)
	if err != nil {
		t.Fatal(err)
	}
	if newPair.RefreshToken == pair.RefreshToken {
		t.Fatalf("rotation must issue new refresh token")
	}
	if _, err := svc.RefreshTokens(context.Background(), pair.RefreshToken); !errors.Is(err, core_errors.ErrUnauthorized) {
		t.Fatalf("reused revoked token should be Unauthorized, got %v", err)
	}
}

type failingRotation struct{ *mockJwtRepo }

func (r *failingRotation) RotateRefreshToken(context.Context, uuid.UUID, *jwt_domain.RefreshToken) error {
	return errors.New("transaction failed")
}
func TestFailedRotationPreservesExistingSession(t *testing.T) {
	svc, repo, users := newEdgeSvc()
	uid := uuid.New()
	users.users[uid] = edgeUser(uid, "rotation")
	pair, err := svc.GeneratePair(context.Background(), uid, "rotation")
	if err != nil {
		t.Fatal(err)
	}
	svc.jwtRepo = &failingRotation{repo}
	if _, err = svc.RefreshTokens(context.Background(), pair.RefreshToken); err == nil {
		t.Fatal("expected transaction failure")
	}
	if len(repo.tokens) != 1 {
		t.Fatal("failed rotation persisted a replacement")
	}
	for _, token := range repo.tokens {
		if token.RevokedAt != nil {
			t.Fatal("failed rotation revoked existing session")
		}
	}
}
