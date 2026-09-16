package http

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	core_errors "github.com/Aam-Shaegar/Rhythm/internal/core/errors"
	"github.com/Aam-Shaegar/Rhythm/internal/features/reports/domain"
	"github.com/google/uuid"
	"time"
)

type mockReportsSvc struct {
	dailyFn  func(ctx context.Context, uid uuid.UUID, d time.Time) (*domain.DailyReport, error)
	periodFn func(ctx context.Context, uid uuid.UUID, from, to time.Time) ([]*domain.DailyReport, error)
}

func (s *mockReportsSvc) GetDailyReport(ctx context.Context, uid uuid.UUID, d time.Time) (*domain.DailyReport, error) {
	return s.dailyFn(ctx, uid, d)
}
func (s *mockReportsSvc) GetPeriodReport(ctx context.Context, uid uuid.UUID, from, to time.Time) ([]*domain.DailyReport, error) {
	return s.periodFn(ctx, uid, from, to)
}

func withUser(r *http.Request, uid string) *http.Request {
	return r.WithContext(context.WithValue(r.Context(), "user_id", uid))
}

func TestReports_Daily_NoUser_401(t *testing.T) {
	h := NewReportsHTTPHandler(&mockReportsSvc{})
	r := httptest.NewRequest("GET", "/reports/daily", nil)
	w := httptest.NewRecorder()
	h.Daily(w, r)
	if w.Code != 401 {
		t.Fatalf("want 401 got %d", w.Code)
	}
}

func TestReports_Daily_BadDate_400(t *testing.T) {
	uid := uuid.New().String()
	h := NewReportsHTTPHandler(&mockReportsSvc{dailyFn: func(ctx context.Context, u uuid.UUID, d time.Time) (*domain.DailyReport, error) {
		t.Fatalf("service must not be called")
		return nil, nil
	}})
	r := httptest.NewRequest("GET", "/reports/daily?date=not-a-date", nil)
	r = withUser(r, uid)
	w := httptest.NewRecorder()
	h.Daily(w, r)
	if w.Code != 400 {
		t.Fatalf("want 400 got %d %s", w.Code, w.Body.String())
	}
}

func TestReports_Daily_Ok_200_SnakeCase(t *testing.T) {
	uid := uuid.New().String()
	h := NewReportsHTTPHandler(&mockReportsSvc{dailyFn: func(ctx context.Context, u uuid.UUID, d time.Time) (*domain.DailyReport, error) {
		return &domain.DailyReport{Date: d, TotalTasks: 2, CompletedTasks: 1, CompletionPct: 50, EventsCount: 3}, nil
	}})
	r := httptest.NewRequest("GET", "/reports/daily", nil)
	r = withUser(r, uid)
	w := httptest.NewRecorder()
	h.Daily(w, r)
	if w.Code != 200 {
		t.Fatalf("want 200 got %d", w.Code)
	}
	body := w.Body.String()
	for _, k := range []string{`"total_tasks"`, `"completed_tasks"`, `"completion_pct"`, `"events_count"`} {
		if !strings.Contains(body, k) {
			t.Fatalf("missing snake_case key %s in %s", k, body)
		}
	}
	if strings.Contains(body, "TotalTasks") {
		t.Fatalf("CamelCase leaked: %s", body)
	}
}

func TestReports_Period_ServiceError_500(t *testing.T) {
	uid := uuid.New().String()
	h := NewReportsHTTPHandler(&mockReportsSvc{
		dailyFn: func(ctx context.Context, u uuid.UUID, d time.Time) (*domain.DailyReport, error) {
			return &domain.DailyReport{}, nil
		},
		periodFn: func(ctx context.Context, u uuid.UUID, from, to time.Time) ([]*domain.DailyReport, error) {
			return nil, context.DeadlineExceeded
		},
	})
	r := httptest.NewRequest("GET", "/reports/period", nil)
	r = withUser(r, uid)
	w := httptest.NewRecorder()
	h.Period(w, r)
	if w.Code != 500 {
		t.Fatalf("want 500 got %d", w.Code)
	}
}

func TestReports_Period_BadFrom_400(t *testing.T) {
	uid := uuid.New().String()
	h := NewReportsHTTPHandler(&mockReportsSvc{})
	r := httptest.NewRequest("GET", "/reports/period?from=bad&to=2026-01-01", nil)
	r = withUser(r, uid)
	w := httptest.NewRecorder()
	h.Period(w, r)
	if w.Code != 400 {
		t.Fatalf("want 400 got %d", w.Code)
	}
}

func TestReports_Daily_ServiceNotFound_404(t *testing.T) {
	uid := uuid.New().String()
	h := NewReportsHTTPHandler(&mockReportsSvc{dailyFn: func(ctx context.Context, u uuid.UUID, d time.Time) (*domain.DailyReport, error) {
		return nil, core_errors.ErrNotFound
	}})
	r := httptest.NewRequest("GET", "/reports/daily", nil)
	r = withUser(r, uid)
	w := httptest.NewRecorder()
	h.Daily(w, r)
	if w.Code != 404 {
		t.Fatalf("want 404 got %d", w.Code)
	}
}
