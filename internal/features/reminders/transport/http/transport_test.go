package http

import (
	"context"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Aam-Shaegar/Rhythm/internal/features/reminders/domain"
	"github.com/Aam-Shaegar/Rhythm/internal/features/reminders/service"
	"github.com/google/uuid"
)

type settingsSvc struct {
	service.RemindersService
	settings domain.ReminderSettings
}

func (s *settingsSvc) GetSettings(context.Context, uuid.UUID) (domain.ReminderSettings, error) {
	return s.settings, nil
}
func (s *settingsSvc) UpdateSettings(_ context.Context, _ uuid.UUID, in domain.UpdateReminderSettingsInput) (domain.ReminderSettings, error) {
	if in.EventOffsets != nil {
		s.settings.EventOffsets = *in.EventOffsets
	}
	if in.TaskOffsets != nil {
		s.settings.TaskOffsets = *in.TaskOffsets
	}
	return s.settings, nil
}

func TestReminders_NoUser_401(t *testing.T) {
	h := NewRemindersHTTPHandler(&settingsSvc{settings: domain.DefaultSettings()})
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
	h := NewRemindersHTTPHandler(&settingsSvc{settings: domain.DefaultSettings()})
	uid := uuid.New().String()
	r := httptest.NewRequest("GET", "/reminders/settings", nil)
	r = r.WithContext(context.WithValue(r.Context(), "user_id", uid))
	w := httptest.NewRecorder()
	h.Settings(w, r)
	if w.Code != 200 {
		t.Fatalf("want 200 got %d %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "event_offsets") {
		t.Fatalf("missing keys: %s", w.Body.String())
	}
	r = httptest.NewRequest("PATCH", "/reminders/settings", strings.NewReader(`{"event_offsets":[15,0]}`))
	r = r.WithContext(context.WithValue(r.Context(), "user_id", uid))
	w = httptest.NewRecorder()
	h.UpdateSettings(w, r)
	if w.Code != 200 {
		t.Fatalf("want 200 got %d %s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), `"event_offsets":[15,0]`) {
		t.Fatalf("patch not reflected: %s", w.Body.String())
	}
}

func TestReminders_Update_BadJSON_400(t *testing.T) {
	h := NewRemindersHTTPHandler(&settingsSvc{settings: domain.DefaultSettings()})
	uid := uuid.New().String()
	r := httptest.NewRequest("PATCH", "/reminders/settings", strings.NewReader(`{bad`))
	r = r.WithContext(context.WithValue(r.Context(), "user_id", uid))
	w := httptest.NewRecorder()
	h.UpdateSettings(w, r)
	if w.Code != 400 {
		t.Fatalf("want 400 got %d", w.Code)
	}
}
