package create

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	core_errors "github.com/Aam-Shaegar/Rhythm/internal/core/errors"
	"github.com/Aam-Shaegar/Rhythm/internal/features/tasks/domain"
	"github.com/google/uuid"
)

type stubTasksSvc struct {
	createFn   func(ctx context.Context, uid uuid.UUID, in domain.CreateTaskInput) (*domain.TaskResponse, error)
	getFn      func(ctx context.Context, uid, id uuid.UUID) (*domain.TaskResponse, error)
	listFn     func(ctx context.Context, uid uuid.UUID, f domain.TaskFilter) ([]*domain.TaskResponse, error)
	updateFn   func(ctx context.Context, uid, id uuid.UUID, in domain.UpdateTaskInput) (*domain.TaskResponse, error)
	completeFn func(ctx context.Context, uid, id uuid.UUID) (*domain.TaskResponse, error)
	deleteFn   func(ctx context.Context, uid, id uuid.UUID) error
}

func (s *stubTasksSvc) CreateTask(ctx context.Context, uid uuid.UUID, in domain.CreateTaskInput) (*domain.TaskResponse, error) {
	return s.createFn(ctx, uid, in)
}
func (s *stubTasksSvc) GetTask(ctx context.Context, uid, id uuid.UUID) (*domain.TaskResponse, error) {
	return s.getFn(ctx, uid, id)
}
func (s *stubTasksSvc) ListTasks(ctx context.Context, uid uuid.UUID, f domain.TaskFilter) ([]*domain.TaskResponse, error) {
	return s.listFn(ctx, uid, f)
}
func (s *stubTasksSvc) UpdateTask(ctx context.Context, uid, id uuid.UUID, in domain.UpdateTaskInput) (*domain.TaskResponse, error) {
	return s.updateFn(ctx, uid, id, in)
}
func (s *stubTasksSvc) CompleteTask(ctx context.Context, uid, id uuid.UUID) (*domain.TaskResponse, error) {
	return s.completeFn(ctx, uid, id)
}
func (s *stubTasksSvc) DeleteTask(ctx context.Context, uid, id uuid.UUID) error {
	return s.deleteFn(ctx, uid, id)
}
func (s *stubTasksSvc) GenerateRecurringTasks(ctx context.Context, before time.Time) error {
	return nil
}

func withUser(r *http.Request, uid string) *http.Request {
	ctx := context.WithValue(r.Context(), "user_id", uid)
	// also typed key used by fixed middleware
	type ctxKey string
	ctx = context.WithValue(ctx, ctxKey("user_id"), uid)
	// middleware package key is its own type; emulate via real middleware helper path:
	// simplest: set both string and use middleware-tested flow by also storing under the typed key
	// via context.WithValue with string works because our fix stores string key too.
	return r.WithContext(ctx)
}

func TestCreate_NoUser_401(t *testing.T) {
	h := NewHandler(&stubTasksSvc{})
	r := httptest.NewRequest("POST", "/tasks", strings.NewReader(`{"title":"t","due_at":"2026-09-20T10:00:00Z"}`))
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != 401 {
		t.Fatalf("expected 401, got %d %s", w.Code, w.Body.String())
	}
}

func TestCreate_InvalidUUID_400(t *testing.T) {
	h := NewHandler(&stubTasksSvc{})
	r := httptest.NewRequest("POST", "/tasks", strings.NewReader(`{"title":"t","due_at":"2026-09-20T10:00:00Z"}`))
	r = withUser(r, "not-a-uuid")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != 400 {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestCreate_BadBody_400(t *testing.T) {
	uid := uuid.New().String()
	for _, body := range []string{
		`{"title":"","due_at":"2026-09-20T10:00:00Z"}`,
		`{"title":"t","due_at":"bad"}`,
		`{bad json`,
		`{}`,
	} {
		h := NewHandler(&stubTasksSvc{createFn: func(ctx context.Context, u uuid.UUID, in domain.CreateTaskInput) (*domain.TaskResponse, error) {
			t.Fatalf("service must not be called on invalid body")
			return nil, nil
		}})
		r := httptest.NewRequest("POST", "/tasks", strings.NewReader(body))
		r = withUser(r, uid)
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		if w.Code != 400 {
			t.Fatalf("body %q: expected 400 got %d %s", body, w.Code, w.Body.String())
		}
	}
}

func TestCreate_ServiceNotFound_404(t *testing.T) {
	uid := uuid.New().String()
	h := NewHandler(&stubTasksSvc{createFn: func(ctx context.Context, u uuid.UUID, in domain.CreateTaskInput) (*domain.TaskResponse, error) {
		return nil, core_errors.ErrNotFound
	}})
	r := httptest.NewRequest("POST", "/tasks", strings.NewReader(`{"title":"t","due_at":"2026-09-20T10:00:00Z"}`))
	r = withUser(r, uid)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != 404 {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func TestCreate_Success_201(t *testing.T) {
	uid := uuid.New()
	h := NewHandler(&stubTasksSvc{createFn: func(ctx context.Context, u uuid.UUID, in domain.CreateTaskInput) (*domain.TaskResponse, error) {
		if u != uid {
			t.Fatalf("wrong uid")
		}
		return &domain.TaskResponse{ID: uuid.New(), Title: in.Title}, nil
	}})
	r := httptest.NewRequest("POST", "/tasks", strings.NewReader(`{"title":"t","due_at":"2026-09-20T10:00:00Z"}`))
	r = withUser(r, uid.String())
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != 201 {
		t.Fatalf("expected 201, got %d %s", w.Code, w.Body.String())
	}
}
