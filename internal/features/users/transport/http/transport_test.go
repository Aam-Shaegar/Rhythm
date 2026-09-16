package http

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	core_config "github.com/Aam-Shaegar/Rhythm/internal/core/config"
	core_errors "github.com/Aam-Shaegar/Rhythm/internal/core/errors"
	core_logger "github.com/Aam-Shaegar/Rhythm/internal/core/logger"
	core_middleware "github.com/Aam-Shaegar/Rhythm/internal/core/transport/http/middleware"
	"github.com/Aam-Shaegar/Rhythm/internal/features/users/domain/dtos"
	"github.com/google/uuid"
)

func testCfg() *core_config.Config {
	return &core_config.Config{JwtRefreshTTL: time.Hour, SecureRefreshCookie: false}
}

type mockUsersSvc struct {
	registerFn func(ctx context.Context, in dtos.RegisterInput) (*dtos.AuthResponse, error)
	loginFn    func(ctx context.Context, in dtos.LoginInput) (*dtos.AuthResponse, error)
	getFn      func(ctx context.Context, uid uuid.UUID) (*dtos.UserResponse, error)
	updateFn   func(ctx context.Context, uid uuid.UUID, in dtos.UpdateProfileInput) (*dtos.UserResponse, error)
}

func (s *mockUsersSvc) Register(ctx context.Context, in dtos.RegisterInput) (*dtos.AuthResponse, error) {
	return s.registerFn(ctx, in)
}
func (s *mockUsersSvc) Login(ctx context.Context, in dtos.LoginInput) (*dtos.AuthResponse, error) {
	return s.loginFn(ctx, in)
}
func (s *mockUsersSvc) GetMe(ctx context.Context, uid uuid.UUID) (*dtos.UserResponse, error) {
	return s.getFn(ctx, uid)
}
func (s *mockUsersSvc) UpdateProfile(ctx context.Context, uid uuid.UUID, in dtos.UpdateProfileInput) (*dtos.UserResponse, error) {
	return s.updateFn(ctx, uid, in)
}

func withUser(r *http.Request, uid string) *http.Request {
	// users handlers use typed middleware key via core_middleware.GetUserID.
	// Go through the real Auth middleware so both typed and legacy string keys are set
	// exactly as in production (regression test for the "user not in context" bug).
	// Auth middleware also needs a logger in context.
	cfg := core_logger.LoggerConfig{Folder: "/tmp", Level: "ERROR"}
	l, _ := core_logger.NewLogger(cfg)
	// logger Close would close file; leak is fine in tests, but close via cleanup pattern is skipped
	// to keep context valid during handler call.
	r = r.WithContext(core_logger.ToContext(r.Context(), l))
	var out *http.Request
	auth := core_middleware.Auth(func(tok string) (string, string, error) {
		return uid, "testuser", nil
	})
	auth(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		out = req
	})).ServeHTTP(httptest.NewRecorder(), func() *http.Request {
		r.Header.Set("Authorization", "Bearer dummy")
		return r
	}())
	if out == nil {
		return r.WithContext(context.WithValue(r.Context(), "user_id", uid))
	}
	return out
}

func TestUsers_Register_BadBody_400(t *testing.T) {
	h := NewUsersHTTPHandler(&mockUsersSvc{registerFn: func(ctx context.Context, in dtos.RegisterInput) (*dtos.AuthResponse, error) {
		t.Fatalf("must not be called")
		return nil, nil
	}}, testCfg())
	for _, body := range []string{
		`{"username":"ab","email":"a@b.com","password":"password123"}`,
		`{"username":"alice","email":"bad","password":"password123"}`,
		`{"username":"alice","email":"a@b.com","password":"123"}`,
		`{}`,
	} {
		r := httptest.NewRequest("POST", "/auth/register", strings.NewReader(body))
		w := httptest.NewRecorder()
		h.Register(w, r)
		if w.Code != 400 {
			t.Fatalf("body %s: want 400 got %d %s", body, w.Code, w.Body.String())
		}
	}
}

func TestUsers_Register_Conflict_409(t *testing.T) {
	h := NewUsersHTTPHandler(&mockUsersSvc{registerFn: func(ctx context.Context, in dtos.RegisterInput) (*dtos.AuthResponse, error) {
		return nil, core_errors.ErrConflict
	}}, testCfg())
	r := httptest.NewRequest("POST", "/auth/register", strings.NewReader(`{"username":"alice","email":"a@b.com","password":"password123"}`))
	w := httptest.NewRecorder()
	h.Register(w, r)
	if w.Code != 409 {
		t.Fatalf("want 409 got %d", w.Code)
	}
}

func TestUsers_Login_BadBody_400(t *testing.T) {
	h := NewUsersHTTPHandler(&mockUsersSvc{loginFn: func(ctx context.Context, in dtos.LoginInput) (*dtos.AuthResponse, error) {
		t.Fatalf("must not be called")
		return nil, nil
	}}, testCfg())
	r := httptest.NewRequest("POST", "/auth/login", strings.NewReader(`{"email":"bad","password":""}`))
	w := httptest.NewRecorder()
	h.Login(w, r)
	if w.Code != 400 {
		t.Fatalf("want 400 got %d", w.Code)
	}
}

func TestUsers_Login_Unauthorized_401(t *testing.T) {
	h := NewUsersHTTPHandler(&mockUsersSvc{loginFn: func(ctx context.Context, in dtos.LoginInput) (*dtos.AuthResponse, error) {
		return nil, core_errors.ErrUnauthorized
	}}, testCfg())
	r := httptest.NewRequest("POST", "/auth/login", strings.NewReader(`{"email":"a@b.com","password":"wrongpass1"}`))
	w := httptest.NewRecorder()
	h.Login(w, r)
	if w.Code != 401 {
		t.Fatalf("want 401 got %d", w.Code)
	}
}

func TestUsers_GetMe_NoAuth_401(t *testing.T) {
	h := NewUsersHTTPHandler(&mockUsersSvc{getFn: func(ctx context.Context, uid uuid.UUID) (*dtos.UserResponse, error) {
		t.Fatalf("must not be called")
		return nil, nil
	}}, testCfg())
	r := httptest.NewRequest("GET", "/users/me", nil)
	w := httptest.NewRecorder()
	h.GetMe(w, r)
	if w.Code != 401 {
		t.Fatalf("want 401 got %d %s", w.Code, w.Body.String())
	}
}

func TestUsers_GetMe_Ok_200(t *testing.T) {
	uid := uuid.New()
	h := NewUsersHTTPHandler(&mockUsersSvc{getFn: func(ctx context.Context, id uuid.UUID) (*dtos.UserResponse, error) {
		if id != uid {
			t.Fatalf("wrong uid %v want %v", id, uid)
		}
		return &dtos.UserResponse{ID: uid, Username: "alice", Email: "a@b.com"}, nil
	}}, testCfg())
	r := httptest.NewRequest("GET", "/users/me", nil)
	r = withUser(r, uid.String())
	w := httptest.NewRecorder()
	h.GetMe(w, r)
	if w.Code != 200 {
		t.Fatalf("want 200 got %d %s", w.Code, w.Body.String())
	}
}

func TestUsers_GetMe_NotFound_404(t *testing.T) {
	uid := uuid.New()
	h := NewUsersHTTPHandler(&mockUsersSvc{getFn: func(ctx context.Context, id uuid.UUID) (*dtos.UserResponse, error) {
		return nil, core_errors.ErrNotFound
	}}, testCfg())
	r := httptest.NewRequest("GET", "/users/me", nil)
	r = withUser(r, uid.String())
	w := httptest.NewRecorder()
	h.GetMe(w, r)
	if w.Code != 404 {
		t.Fatalf("want 404 got %d", w.Code)
	}
}

func TestUsers_UpdateProfile_BadEmail_400(t *testing.T) {
	uid := uuid.New()
	h := NewUsersHTTPHandler(&mockUsersSvc{updateFn: func(ctx context.Context, id uuid.UUID, in dtos.UpdateProfileInput) (*dtos.UserResponse, error) {
		t.Fatalf("must not be called")
		return nil, nil
	}}, testCfg())
	r := httptest.NewRequest("PATCH", "/users/me", strings.NewReader(`{"email":"bad"}`))
	r = withUser(r, uid.String())
	w := httptest.NewRecorder()
	h.UpdateProfile(w, r)
	if w.Code != 400 {
		t.Fatalf("want 400 got %d %s", w.Code, w.Body.String())
	}
}

func TestUsers_UpdateProfile_Conflict_409(t *testing.T) {
	uid := uuid.New()
	h := NewUsersHTTPHandler(&mockUsersSvc{updateFn: func(ctx context.Context, id uuid.UUID, in dtos.UpdateProfileInput) (*dtos.UserResponse, error) {
		return nil, core_errors.ErrConflict
	}}, testCfg())
	r := httptest.NewRequest("PATCH", "/users/me", strings.NewReader(`{"username":"bob"}`))
	r = withUser(r, uid.String())
	w := httptest.NewRecorder()
	h.UpdateProfile(w, r)
	if w.Code != 409 {
		t.Fatalf("want 409 got %d", w.Code)
	}
}
