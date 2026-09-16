package http

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	core_errors "github.com/Aam-Shaegar/Rhythm/internal/core/errors"
	"github.com/Aam-Shaegar/Rhythm/internal/features/events/domain"
	"github.com/google/uuid"
)

type mockSvc struct {
	createFn func(ctx context.Context, uid uuid.UUID, in domain.CreateEventInput) (*domain.EventResponse, error)
	getFn    func(ctx context.Context, uid, id uuid.UUID) (*domain.EventResponse, error)
	listFn   func(ctx context.Context, uid uuid.UUID, f domain.EventFilter) ([]*domain.EventResponse, error)
	updateFn func(ctx context.Context, uid, id uuid.UUID, in domain.UpdateEventInput) (*domain.EventResponse, error)
	deleteFn func(ctx context.Context, uid, id uuid.UUID) error
}

func (s *mockSvc) CreateEvent(ctx context.Context, uid uuid.UUID, in domain.CreateEventInput) (*domain.EventResponse, error) {
	return s.createFn(ctx, uid, in)
}
func (s *mockSvc) GetEvent(ctx context.Context, uid, id uuid.UUID) (*domain.EventResponse, error) {
	return s.getFn(ctx, uid, id)
}
func (s *mockSvc) ListEvents(ctx context.Context, uid uuid.UUID, f domain.EventFilter) ([]*domain.EventResponse, error) {
	return s.listFn(ctx, uid, f)
}
func (s *mockSvc) UpdateEvent(ctx context.Context, uid, id uuid.UUID, in domain.UpdateEventInput) (*domain.EventResponse, error) {
	return s.updateFn(ctx, uid, id, in)
}
func (s *mockSvc) DeleteEvent(ctx context.Context, uid, id uuid.UUID) error {
	return s.deleteFn(ctx, uid, id)
}

func withUser(r *http.Request, uid string) *http.Request {
	return r.WithContext(context.WithValue(r.Context(), "user_id", uid))
}

func TestEvents_Create_EndBeforeStart_400(t *testing.T) {
	// validation of end>start lives in service; transport must surface 400 for bad body too.
	// Here: malformed datetime -> 400 without calling service.
	uid := uuid.New().String()
	h := NewEventsHTTPHandler(&mockSvc{createFn: func(ctx context.Context, u uuid.UUID, in domain.CreateEventInput) (*domain.EventResponse, error) {
		t.Fatalf("service must not be called")
		return nil, nil
	}})
	for _, body := range []string{
		`{"title":"t","start_at":"bad","end_at":"2026-09-20T11:00:00Z"}`,
		`{"title":"","start_at":"2026-09-20T10:00:00Z","end_at":"2026-09-20T11:00:00Z"}`,
		`{}`,
	} {
		r := httptest.NewRequest("POST", "/events", strings.NewReader(body))
		r = withUser(r, uid)
		w := httptest.NewRecorder()
		h.Create(w, r)
		if w.Code != 400 {
			t.Fatalf("body %s: want 400 got %d", body, w.Code)
		}
	}
}

func TestEvents_Get_EdgeCases(t *testing.T) {
	uid := uuid.New()
	eid := uuid.New()
	h := NewEventsHTTPHandler(&mockSvc{getFn: func(ctx context.Context, u, id uuid.UUID) (*domain.EventResponse, error) {
		if id == eid {
			return &domain.EventResponse{ID: eid}, nil
		}
		return nil, core_errors.ErrNotFound
	}})
	r := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	h.Get(w, r)
	if w.Code != 401 {
		t.Fatalf("no user want 401 got %d", w.Code)
	}
	r = httptest.NewRequest("GET", "/", nil)
	r = withUser(r, uid.String())
	r.SetPathValue("id", "bad")
	w = httptest.NewRecorder()
	h.Get(w, r)
	if w.Code != 400 {
		t.Fatalf("bad id want 400 got %d", w.Code)
	}
	r = httptest.NewRequest("GET", "/", nil)
	r = withUser(r, uid.String())
	r.SetPathValue("id", eid.String())
	w = httptest.NewRecorder()
	h.Get(w, r)
	if w.Code != 200 {
		t.Fatalf("ok want 200 got %d", w.Code)
	}
}

func TestEvents_Delete_404_204(t *testing.T) {
	uid := uuid.New()
	h := NewEventsHTTPHandler(&mockSvc{deleteFn: func(ctx context.Context, u, id uuid.UUID) error {
		return core_errors.ErrNotFound
	}})
	r := httptest.NewRequest("DELETE", "/", nil)
	r = withUser(r, uid.String())
	r.SetPathValue("id", uuid.NewString())
	w := httptest.NewRecorder()
	h.Delete(w, r)
	if w.Code != 404 {
		t.Fatalf("want 404 got %d", w.Code)
	}
	h2 := NewEventsHTTPHandler(&mockSvc{deleteFn: func(ctx context.Context, u, id uuid.UUID) error { return nil }})
	r = httptest.NewRequest("DELETE", "/", nil)
	r = withUser(r, uid.String())
	r.SetPathValue("id", uuid.NewString())
	w = httptest.NewRecorder()
	h2.Delete(w, r)
	if w.Code != 204 {
		t.Fatalf("want 204 got %d", w.Code)
	}
}

func TestEvents_List_BadView_400(t *testing.T) {
	uid := uuid.New()
	h := NewEventsHTTPHandler(&mockSvc{listFn: func(ctx context.Context, u uuid.UUID, f domain.EventFilter) ([]*domain.EventResponse, error) {
		return []*domain.EventResponse{}, nil
	}})
	r := httptest.NewRequest("GET", "/events?view=year", nil)
	r = withUser(r, uid.String())
	w := httptest.NewRecorder()
	h.List(w, r)
	if w.Code != 400 {
		t.Fatalf("bad view want 400 got %d %s", w.Code, w.Body.String())
	}
}
