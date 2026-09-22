package http

import (
	"context"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Aam-Shaegar/Rhythm/internal/features/reminders/domain"
	"github.com/google/uuid"
)

type stubRemindersSvc struct{}

func (s *stubRemindersSvc) ScheduleForEvent(ctx context.Context, e interface{}, u uuid.UUID) error {
	return nil
}
func (s *stubRemindersSvc) ScheduleForTask(ctx context.Context, t interface{}, u uuid.UUID) error {
	return nil
}
func (s *stubRemindersSvc) UpdateRemindersForEvent(ctx context.Context, e interface{}, u uuid.UUID) error {
	return nil
}
func (s *stubRemindersSvc) UpdateRemindersForTask(ctx context.Context, t interface{}, u uuid.UUID) error {
	return nil
}
func (s *stubRemindersSvc) DeleteRemindersForEvent(ctx context.Context, id uuid.UUID) error {
	return nil
}
func (s *stubRemindersSvc) DeleteRemindersForTask(ctx context.Context, id uuid.UUID) error {
	return nil
}
func (s *stubRemindersSvc) ProcessPendingReminders(ctx context.Context, b interface{}, l int) (int, error) {
	return 0, nil
}
func (s *stubRemindersSvc) SavePushSubscription(ctx context.Context, u uuid.UUID, in domain.PushSubscriptionInput) error {
	return nil
}
func (s *stubRemindersSvc) DeletePushSubscription(ctx context.Context, u uuid.UUID, endpoint string) error {
	return nil
}
func (s *stubRemindersSvc) PushPublicKey() string {
	return "test-public-key"
}

// NOTE: RemindersHTTPHandler requires service.RemindersService with concrete
// event/task types; instead test the settings handlers directly via routes with a nil svc
// because they don't touch the service (stub behavior). Auth is the edge under test.

func TestReminders_NoUser_401(t *testing.T) {
	h := NewRemindersHTTPHandler(nil)
	r := httptest.NewRequest("GET", "/reminders/settings", nil)
	w := httptest.NewRecorder()
	h.Settings(w, r)
	if w.Code != 401 {
		t.Fatalf("want 401 got %d %s", w.Code, w.Body.String())
	}
	r = httptest.NewRequest("PATCH", "/reminders/settings", strings.NewReader(`{}`))
	w = httptest.NewRecorder()
	h.UpdateSettings(w, r)
	if w.Code != 401 {
		t.Fatalf("want 401 got %d", w.Code)
	}
}

func TestReminders_WithUser_200(t *testing.T) {
	h := NewRemindersHTTPHandler(nil)
	uid := uuid.New().String()
	r := httptest.NewRequest("GET", "/reminders/settings", nil)
	r = r.WithContext(context.WithValue(r.Context(), "user_id", uid))
	w := httptest.NewRecorder()
	h.Settings(w, r)
	if w.Code != 200 {
		t.Fatalf("want 200 got %d %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "event_reminders") {
		t.Fatalf("missing keys: %s", w.Body.String())
	}
	r = httptest.NewRequest("PATCH", "/reminders/settings", strings.NewReader(`{"event_reminders":false}`))
	r = r.WithContext(context.WithValue(r.Context(), "user_id", uid))
	w = httptest.NewRecorder()
	h.UpdateSettings(w, r)
	if w.Code != 200 {
		t.Fatalf("want 200 got %d %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), `"event_reminders":false`) {
		t.Fatalf("patch not reflected: %s", w.Body.String())
	}
}

func TestReminders_Update_BadJSON_400(t *testing.T) {
	h := NewRemindersHTTPHandler(nil)
	uid := uuid.New().String()
	r := httptest.NewRequest("PATCH", "/reminders/settings", strings.NewReader(`{bad`))
	r = r.WithContext(context.WithValue(r.Context(), "user_id", uid))
	w := httptest.NewRecorder()
	h.UpdateSettings(w, r)
	if w.Code != 400 {
		t.Fatalf("want 400 got %d", w.Code)
	}
}
