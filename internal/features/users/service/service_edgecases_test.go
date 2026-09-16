package service

import (
	"context"
	"errors"
	"testing"

	core_errors "github.com/Aam-Shaegar/Rhythm/internal/core/errors"
	"github.com/Aam-Shaegar/Rhythm/internal/features/users/domain/dtos"
	"github.com/google/uuid"
)

func TestRegister_DuplicateUsername(t *testing.T) {
	repo := newMockUsersRepo()
	svc := NewUsersService(repo, newMockJwtService())
	_, err := svc.Register(context.Background(), dtos.RegisterInput{Username: "bob", Email: "bob@ex.com", Password: "password123"})
	if err != nil {
		t.Fatal(err)
	}
	_, err = svc.Register(context.Background(), dtos.RegisterInput{Username: "bob", Email: "other@ex.com", Password: "password123"})
	if !errors.Is(err, core_errors.ErrConflict) {
		t.Fatalf("duplicate username should be Conflict, got %v", err)
	}
}

func TestLogin_NotFound_MapsToUnauthorized(t *testing.T) {
	svc := NewUsersService(newMockUsersRepo(), newMockJwtService())
	_, err := svc.Login(context.Background(), dtos.LoginInput{Email: "ghost@ex.com", Password: "whatever123"})
	if !errors.Is(err, core_errors.ErrUnauthorized) {
		t.Fatalf("unknown email should be Unauthorized (not leak existence), got %v", err)
	}
}

func TestUpdateProfile_ConflictEmail(t *testing.T) {
	repo := newMockUsersRepo()
	svc := NewUsersService(repo, newMockJwtService())
	a, _ := svc.Register(context.Background(), dtos.RegisterInput{Username: "alice", Email: "alice@ex.com", Password: "password123"})
	_, _ = svc.Register(context.Background(), dtos.RegisterInput{Username: "bob", Email: "bob@ex.com", Password: "password123"})
	bobEmail := "alice@ex.com"
	_, err := svc.UpdateProfile(context.Background(), a.User.ID, dtos.UpdateProfileInput{Email: &bobEmail})
	// updating alice to bob's... wait bob is second; alice taking bob's email? use bob's email:
	_ = bobEmail
	bEmail := "bob@ex.com"
	_, err = svc.UpdateProfile(context.Background(), a.User.ID, dtos.UpdateProfileInput{Email: &bEmail})
	if !errors.Is(err, core_errors.ErrConflict) {
		t.Fatalf("email conflict should be Conflict, got %v", err)
	}
}

func TestUpdateProfile_ConflictUsername(t *testing.T) {
	repo := newMockUsersRepo()
	svc := NewUsersService(repo, newMockJwtService())
	a, _ := svc.Register(context.Background(), dtos.RegisterInput{Username: "alice", Email: "alice@ex.com", Password: "password123"})
	_, _ = svc.Register(context.Background(), dtos.RegisterInput{Username: "bob", Email: "bob@ex.com", Password: "password123"})
	bName := "bob"
	_, err := svc.UpdateProfile(context.Background(), a.User.ID, dtos.UpdateProfileInput{Username: &bName})
	if !errors.Is(err, core_errors.ErrConflict) {
		t.Fatalf("expected Conflict, got %v", err)
	}
}

func TestUpdateProfile_SameValues_NoConflict(t *testing.T) {
	repo := newMockUsersRepo()
	svc := NewUsersService(repo, newMockJwtService())
	a, _ := svc.Register(context.Background(), dtos.RegisterInput{Username: "alice", Email: "alice@ex.com", Password: "password123"})
	sameU := "alice"
	sameE := "alice@ex.com"
	updated, err := svc.UpdateProfile(context.Background(), a.User.ID, dtos.UpdateProfileInput{Username: &sameU, Email: &sameE})
	if err != nil {
		t.Fatalf("same values should not conflict: %v", err)
	}
	if updated.Username != "alice" {
		t.Fatalf("wrong username %q", updated.Username)
	}
}

func TestUpdateProfile_EmptyUpdate_KeepsValues(t *testing.T) {
	repo := newMockUsersRepo()
	svc := NewUsersService(repo, newMockJwtService())
	a, _ := svc.Register(context.Background(), dtos.RegisterInput{Username: "alice", Email: "alice@ex.com", Password: "password123"})
	updated, err := svc.UpdateProfile(context.Background(), a.User.ID, dtos.UpdateProfileInput{})
	if err != nil {
		t.Fatal(err)
	}
	if updated.Username != "alice" || updated.Email != "alice@ex.com" {
		t.Fatalf("empty update changed values: %+v", updated)
	}
}

func TestUpdateProfile_NotFound(t *testing.T) {
	svc := NewUsersService(newMockUsersRepo(), newMockJwtService())
	n := "x"
	_, err := svc.UpdateProfile(context.Background(), uuid.New(), dtos.UpdateProfileInput{Username: &n})
	if !errors.Is(err, core_errors.ErrNotFound) {
		t.Fatalf("expected NotFound, got %v", err)
	}
}

func TestGetMe_NotFound_Edge(t *testing.T) {
	svc := NewUsersService(newMockUsersRepo(), newMockJwtService())
	_, err := svc.GetMe(context.Background(), uuid.New())
	if !errors.Is(err, core_errors.ErrNotFound) {
		t.Fatalf("expected NotFound, got %v", err)
	}
}
