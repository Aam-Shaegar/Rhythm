package http

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	core_errors "github.com/Aam-Shaegar/Rhythm/internal/core/errors"
	jwt_domain "github.com/Aam-Shaegar/Rhythm/internal/features/jwt/domain"
	"github.com/Aam-Shaegar/Rhythm/internal/features/jwt/service"
	"github.com/google/uuid"
)

type mockJwtSvc struct {
	refreshFn func(ctx context.Context, tok string) (*jwt_domain.TokenPair, error)
}

func (s *mockJwtSvc) GeneratePair(ctx context.Context, uid uuid.UUID, u string) (*jwt_domain.TokenPair, error) {
	return nil, nil
}
func (s *mockJwtSvc) ValidateAccessToken(tok string) (string, string, error) { return "", "", nil }
func (s *mockJwtSvc) RefreshTokens(ctx context.Context, tok string) (*jwt_domain.TokenPair, error) {
	return s.refreshFn(ctx, tok)
}
func (s *mockJwtSvc) RevokeRefreshToken(ctx context.Context, tok string) error            { return nil }
func (s *mockJwtSvc) StartCleanup(ctx context.Context, d time.Duration, l service.Logger) {}

func TestRefresh_MissingToken_401(t *testing.T) {
	h := NewJwtHTTPHandler(&mockJwtSvc{}, time.Hour, false)
	r := httptest.NewRequest("POST", "/auth/refresh", nil)
	w := httptest.NewRecorder()
	h.Refresh(w, r)
	if w.Code != 401 {
		t.Fatalf("want 401 got %d %s", w.Code, w.Body.String())
	}
}

func TestRefresh_Invalid_401(t *testing.T) {
	h := NewJwtHTTPHandler(&mockJwtSvc{refreshFn: func(ctx context.Context, tok string) (*jwt_domain.TokenPair, error) {
		return nil, core_errors.ErrUnauthorized
	}}, time.Hour, false)
	r := httptest.NewRequest("POST", "/auth/refresh", nil)
	r.Header.Set("X-Refresh-Token", "bad")
	w := httptest.NewRecorder()
	h.Refresh(w, r)
	if w.Code != 401 {
		t.Fatalf("want 401 got %d", w.Code)
	}
}

func TestRefresh_ViaCookie_Success_200(t *testing.T) {
	h := NewJwtHTTPHandler(&mockJwtSvc{refreshFn: func(ctx context.Context, tok string) (*jwt_domain.TokenPair, error) {
		if tok != "cookie-token" {
			t.Fatalf("expected cookie token, got %q", tok)
		}
		return &jwt_domain.TokenPair{AccessToken: "new-access", RefreshToken: "new-refresh"}, nil
	}}, time.Hour, false)
	r := httptest.NewRequest("POST", "/auth/refresh", nil)
	r.AddCookie(&http.Cookie{Name: "refresh_token", Value: "cookie-token"})
	w := httptest.NewRecorder()
	h.Refresh(w, r)
	if w.Code != 200 {
		t.Fatalf("want 200 got %d %s", w.Code, w.Body.String())
	}
	if w.Header().Get("Set-Cookie") == "" {
		t.Fatalf("expected Set-Cookie")
	}
}

func TestRefresh_ViaHeader_Success_200(t *testing.T) {
	h := NewJwtHTTPHandler(&mockJwtSvc{refreshFn: func(ctx context.Context, tok string) (*jwt_domain.TokenPair, error) {
		return &jwt_domain.TokenPair{AccessToken: "a", RefreshToken: "r"}, nil
	}}, time.Hour, true)
	r := httptest.NewRequest("POST", "/auth/refresh", nil)
	r.Header.Set("X-Refresh-Token", "hdr-token")
	w := httptest.NewRecorder()
	h.Refresh(w, r)
	if w.Code != 200 {
		t.Fatalf("want 200 got %d", w.Code)
	}
	cookie := w.Header().Get("Set-Cookie")
	if cookie == "" || !contains(cookie, "Secure") {
		t.Fatalf("secure cookie flag missing: %q", cookie)
	}
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (func() bool {
		for i := 0; i+len(sub) <= len(s); i++ {
			if s[i:i+len(sub)] == sub {
				return true
			}
		}
		return false
	})()
}
