package http

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

type mockSvc struct {
	createFn   func(ctx context.Context, uid uuid.UUID, in domain.CreateTaskInput) (*domain.TaskResponse, error)
	getFn      func(ctx context.Context, uid, id uuid.UUID) (*domain.TaskResponse, error)
	listFn     func(ctx context.Context, uid uuid.UUID, f domain.TaskFilter) ([]*domain.TaskResponse, error)
	updateFn   func(ctx context.Context, uid, id uuid.UUID, in domain.UpdateTaskInput) (*domain.TaskResponse, error)
	completeFn func(ctx context.Context, uid, id uuid.UUID) (*domain.TaskResponse, error)
	deleteFn   func(ctx context.Context, uid, id uuid.UUID) error
}

func (s *mockSvc) CreateTask(ctx context.Context, uid uuid.UUID, in domain.CreateTaskInput) (*domain.TaskResponse, error) {
	return s.createFn(ctx, uid, in)
}
func (s *mockSvc) GetTask(ctx context.Context, uid, id uuid.UUID) (*domain.TaskResponse, error) {
	return s.getFn(ctx, uid, id)
}
func (s *mockSvc) ListTasks(ctx context.Context, uid uuid.UUID, f domain.TaskFilter) ([]*domain.TaskResponse, error) {
	return s.listFn(ctx, uid, f)
}
func (s *mockSvc) UpdateTask(ctx context.Context, uid, id uuid.UUID, in domain.UpdateTaskInput) (*domain.TaskResponse, error) {
	return s.updateFn(ctx, uid, id, in)
}
func (s *mockSvc) CompleteTask(ctx context.Context, uid, id uuid.UUID) (*domain.TaskResponse, error) {
	return s.completeFn(ctx, uid, id)
}
func (s *mockSvc) DeleteTask(ctx context.Context, uid, id uuid.UUID) error {
	return s.deleteFn(ctx, uid, id)
}
func (s *mockSvc) GenerateRecurringTasks(ctx context.Context, before time.Time) error { return nil }

func withUser(r *http.Request, uid string) *http.Request {
	return r.WithContext(context.WithValue(r.Context(), "user_id", uid))
}

func TestTransport_Get_EdgeCases(t *testing.T) {
	uid := uuid.New()
	tid := uuid.New()
	h := NewTasksHTTPHandler(&mockSvc{getFn: func(ctx context.Context, u, id uuid.UUID) (*domain.TaskResponse, error) {
		if id == tid {
			return &domain.TaskResponse{ID: tid}, nil
		}
		return nil, core_errors.ErrNotFound
	}})
	// no user -> 401
	r := httptest.NewRequest("GET", "/tasks/"+tid.String(), nil)
	w := httptest.NewRecorder()
	h.Get(w, r)
	if w.Code != 401 {
		t.Fatalf("no user: want 401 got %d", w.Code)
	}
	// bad task uuid -> 400
	r = httptest.NewRequest("GET", "/tasks/bad", nil)
	r = withUser(r, uid.String())
	r.SetPathValue("id", "bad")
	w = httptest.NewRecorder()
	h.Get(w, r)
	if w.Code != 400 {
		t.Fatalf("bad id: want 400 got %d", w.Code)
	}
	// not found -> 404
	r = httptest.NewRequest("GET", "/tasks/"+uuid.NewString(), nil)
	r = withUser(r, uid.String())
	r.SetPathValue("id", uuid.NewString())
	w = httptest.NewRecorder()
	h.Get(w, r)
	if w.Code != 404 {
		t.Fatalf("missing: want 404 got %d", w.Code)
	}
	// success -> 200
	r = httptest.NewRequest("GET", "/tasks/"+tid.String(), nil)
	r = withUser(r, uid.String())
	r.SetPathValue("id", tid.String())
	w = httptest.NewRecorder()
	h.Get(w, r)
	if w.Code != 200 {
		t.Fatalf("ok: want 200 got %d %s", w.Code, w.Body.String())
	}
}

func TestTransport_List_BadQuery_400(t *testing.T) {
	uid := uuid.New()
	h := NewTasksHTTPHandler(&mockSvc{listFn: func(ctx context.Context, u uuid.UUID, f domain.TaskFilter) ([]*domain.TaskResponse, error) {
		return []*domain.TaskResponse{}, nil
	}})
	r := httptest.NewRequest("GET", "/tasks?completed=maybe", nil)
	r = withUser(r, uid.String())
	w := httptest.NewRecorder()
	h.List(w, r)
	if w.Code != 400 {
		t.Fatalf("bad query: want 400 got %d %s", w.Code, w.Body.String())
	}
	// valid filter passes through
	r = httptest.NewRequest("GET", "/tasks?completed=true&limit=5", nil)
	r = withUser(r, uid.String())
	w = httptest.NewRecorder()
	h.List(w, r)
	if w.Code != 200 {
		t.Fatalf("valid filter: want 200 got %d", w.Code)
	}
}

func TestTransport_Update_BadBody_400(t *testing.T) {
	uid := uuid.New()
	tid := uuid.New()
	h := NewTasksHTTPHandler(&mockSvc{updateFn: func(ctx context.Context, u, id uuid.UUID, in domain.UpdateTaskInput) (*domain.TaskResponse, error) {
		t.Fatalf("service must not be called")
		return nil, nil
	}})
	r := httptest.NewRequest("PATCH", "/tasks/"+tid.String(), strings.NewReader(`{"title":""}`))
	r = withUser(r, uid.String())
	r.SetPathValue("id", tid.String())
	w := httptest.NewRecorder()
	h.Update(w, r)
	// empty title with omitempty+min=1: "" fails min -> 400. Note: {"title":""} -> title="" (non-nil, empty) -> invalid.
	if w.Code != 400 {
		t.Fatalf("want 400 got %d %s", w.Code, w.Body.String())
	}
}

func TestTransport_Complete_NotFound_404(t *testing.T) {
	uid := uuid.New()
	h := NewTasksHTTPHandler(&mockSvc{completeFn: func(ctx context.Context, u, id uuid.UUID) (*domain.TaskResponse, error) {
		return nil, core_errors.ErrNotFound
	}})
	r := httptest.NewRequest("PATCH", "/tasks/"+uuid.NewString()+"/complete", nil)
	r = withUser(r, uid.String())
	r.SetPathValue("id", uuid.NewString())
	w := httptest.NewRecorder()
	h.Complete(w, r)
	if w.Code != 404 {
		t.Fatalf("want 404 got %d", w.Code)
	}
}

func TestTransport_Delete_Success_204(t *testing.T) {
	uid := uuid.New()
	tid := uuid.New()
	h := NewTasksHTTPHandler(&mockSvc{deleteFn: func(ctx context.Context, u, id uuid.UUID) error { return nil }})
	r := httptest.NewRequest("DELETE", "/tasks/"+tid.String(), nil)
	r = withUser(r, uid.String())
	r.SetPathValue("id", tid.String())
	w := httptest.NewRecorder()
	h.Delete(w, r)
	if w.Code != 204 {
		t.Fatalf("want 204 got %d", w.Code)
	}
}

func TestTransport_Delete_BadID_400(t *testing.T) {
	uid := uuid.New()
	h := NewTasksHTTPHandler(&mockSvc{deleteFn: func(ctx context.Context, u, id uuid.UUID) error { return nil }})
	r := httptest.NewRequest("DELETE", "/tasks/bad", nil)
	r = withUser(r, uid.String())
	r.SetPathValue("id", "bad")
	w := httptest.NewRecorder()
	h.Delete(w, r)
	if w.Code != 400 {
		t.Fatalf("want 400 got %d", w.Code)
	}
}
