package push

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Aam-Shaegar/Rhythm/internal/features/reminders/domain"
	"github.com/Aam-Shaegar/Rhythm/internal/features/reminders/service"
	events_domain "github.com/Aam-Shaegar/Rhythm/internal/features/events/domain"
	tasks_domain "github.com/Aam-Shaegar/Rhythm/internal/features/tasks/domain"
	"github.com/google/uuid"
)

type stubSvc struct {
	key string
}

func (s *stubSvc) ScheduleForEvent(ctx context.Context, e *events_domain.Event, u uuid.UUID) error {
	return nil
}
func (s *stubSvc) ScheduleForTask(ctx context.Context, t *tasks_domain.Task, u uuid.UUID) error {
	return nil
}
func (s *stubSvc) UpdateRemindersForEvent(ctx context.Context, e *events_domain.Event, u uuid.UUID) error {
	return nil
}
func (s *stubSvc) UpdateRemindersForTask(ctx context.Context, t *tasks_domain.Task, u uuid.UUID) error {
	return nil
}
func (s *stubSvc) DeleteRemindersForEvent(ctx context.Context, id uuid.UUID) error { return nil }
func (s *stubSvc) DeleteRemindersForTask(ctx context.Context, id uuid.UUID) error  { return nil }
func (s *stubSvc) ProcessPendingReminders(ctx context.Context, b time.Time, l int) (int, error) {
	return 0, nil
}
func (s *stubSvc) SavePushSubscription(ctx context.Context, u uuid.UUID, in domain.PushSubscriptionInput) error {
	return nil
}
func (s *stubSvc) DeletePushSubscription(ctx context.Context, u uuid.UUID, endpoint string) error {
	return nil
}
func (s *stubSvc) PushPublicKey() string { return s.key }
func (s *stubSvc) StartWorker(ctx context.Context, d time.Duration, l service.Logger) {}

func withUser(r *http.Request, uid string) *http.Request {
	return r.WithContext(context.WithValue(r.Context(), "user_id", uid))
}

func TestSubscribe_NoUser_401(t *testing.T) {
	h := NewSubscribeHandler(&stubSvc{})
	r := httptest.NewRequest("POST", "/reminders/push/subscriptions", strings.NewReader(`{}`))
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != 401 {
		t.Fatalf("want 401 got %d", w.Code)
	}
}

func TestSubscribe_BadBody_400(t *testing.T) {
	h := NewSubscribeHandler(&stubSvc{})
	for _, body := range []string{
		`{}`,
		`{"endpoint":"not-a-url","p256dh":"1234567890ab","auth":"1234567890ab"}`,
		`{"endpoint":"https://push/x","p256dh":"short","auth":"1234567890ab"}`,
		`{bad`,
	} {
		r := httptest.NewRequest("POST", "/reminders/push/subscriptions", strings.NewReader(body))
		r = withUser(r, uuid.New().String())
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		if w.Code != 400 {
			t.Fatalf("body %q: want 400 got %d %s", body, w.Code, w.Body.String())
		}
	}
}

func TestSubscribe_Ok_200(t *testing.T) {
	h := NewSubscribeHandler(&stubSvc{})
	body := `{"endpoint":"https://push.example/x","p256dh":"1234567890abcdef","auth":"1234567890abcdef"}`
	r := httptest.NewRequest("POST", "/reminders/push/subscriptions", strings.NewReader(body))
	r = withUser(r, uuid.New().String())
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != 200 {
		t.Fatalf("want 200 got %d %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "subscribed") {
		t.Fatalf("unexpected body %s", w.Body.String())
	}
}

func TestUnsubscribe_Ok_200(t *testing.T) {
	h := NewUnsubscribeHandler(&stubSvc{})
	body := `{"endpoint":"https://push.example/x"}`
	r := httptest.NewRequest("DELETE", "/reminders/push/subscriptions", strings.NewReader(body))
	r = withUser(r, uuid.New().String())
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != 200 {
		t.Fatalf("want 200 got %d %s", w.Code, w.Body.String())
	}
}

func TestVapidKey_Ok_And_Missing(t *testing.T) {
	h := NewVapidKeyHandler(&stubSvc{key: "B-public-key"})
	r := httptest.NewRequest("GET", "/reminders/push/vapid-key", nil)
	r = withUser(r, uuid.New().String())
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != 200 || !strings.Contains(w.Body.String(), "B-public-key") {
		t.Fatalf("want key, got %d %s", w.Code, w.Body.String())
	}

	h2 := NewVapidKeyHandler(&stubSvc{key: ""})
	r2 := httptest.NewRequest("GET", "/reminders/push/vapid-key", nil)
	r2 = withUser(r2, uuid.New().String())
	w2 := httptest.NewRecorder()
	h2.ServeHTTP(w2, r2)
	if w2.Code != 404 {
		t.Fatalf("unconfigured push must be 404, got %d", w2.Code)
	}
}
