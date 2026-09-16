package service

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	events_domain "github.com/Aam-Shaegar/Rhythm/internal/features/events/domain"
	"github.com/Aam-Shaegar/Rhythm/internal/features/reports/domain"
	tasks_domain "github.com/Aam-Shaegar/Rhythm/internal/features/tasks/domain"
	"github.com/google/uuid"
)

func mustDaily() *domain.DailyReport { return &domain.DailyReport{} }

type mockTasksRepo struct {
	tasks []*tasks_domain.Task
	err   error
}

func (m *mockTasksRepo) Create(ctx context.Context, t *tasks_domain.Task) error { return nil }
func (m *mockTasksRepo) GetByID(ctx context.Context, id uuid.UUID) (*tasks_domain.Task, error) {
	return nil, nil
}
func (m *mockTasksRepo) GetByUserID(ctx context.Context, f tasks_domain.TaskFilter) ([]*tasks_domain.Task, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.tasks, nil
}
func (m *mockTasksRepo) GetByParentID(ctx context.Context, id uuid.UUID) ([]*tasks_domain.Task, error) {
	return nil, nil
}
func (m *mockTasksRepo) GetRecurringParents(ctx context.Context, u uuid.UUID, b time.Time) ([]*tasks_domain.Task, error) {
	return nil, nil
}
func (m *mockTasksRepo) GetUsersWithRecurringTasks(ctx context.Context, b time.Time) ([]uuid.UUID, error) {
	return nil, nil
}
func (m *mockTasksRepo) Update(ctx context.Context, t *tasks_domain.Task) error { return nil }
func (m *mockTasksRepo) Delete(ctx context.Context, id uuid.UUID) error         { return nil }

type mockEventsRepo struct {
	events []*events_domain.Event
	err    error
}

func (m *mockEventsRepo) Create(ctx context.Context, e *events_domain.Event) error { return nil }
func (m *mockEventsRepo) GetByID(ctx context.Context, id uuid.UUID) (*events_domain.Event, error) {
	return nil, nil
}
func (m *mockEventsRepo) GetByUserID(ctx context.Context, f events_domain.EventFilter) ([]*events_domain.Event, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.events, nil
}
func (m *mockEventsRepo) Update(ctx context.Context, e *events_domain.Event) error { return nil }
func (m *mockEventsRepo) Delete(ctx context.Context, id uuid.UUID) error           { return nil }
func (m *mockEventsRepo) GetUpcoming(ctx context.Context, u uuid.UUID, b time.Time, l int) ([]*events_domain.Event, error) {
	return nil, nil
}

func TestDailyReport_Empty(t *testing.T) {
	svc := NewReportsService(&mockTasksRepo{}, &mockEventsRepo{})
	r, err := svc.GetDailyReport(context.Background(), uuid.New(), time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if r.TotalTasks != 0 || r.CompletedTasks != 0 || r.CompletionPct != 0 || r.EventsCount != 0 {
		t.Fatalf("empty report should be zeros: %+v", r)
	}
}

func TestDailyReport_Partial(t *testing.T) {
	tr := &mockTasksRepo{tasks: []*tasks_domain.Task{{IsCompleted: true}, {IsCompleted: true}, {IsCompleted: false}}}
	er := &mockEventsRepo{events: []*events_domain.Event{{}, {}}}
	svc := NewReportsService(tr, er)
	r, err := svc.GetDailyReport(context.Background(), uuid.New(), time.Date(2026, 9, 16, 15, 4, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	if r.TotalTasks != 3 || r.CompletedTasks != 2 || r.EventsCount != 2 {
		t.Fatalf("wrong counts: %+v", r)
	}
	want := float64(2) / float64(3) * 100
	if r.CompletionPct != want {
		t.Fatalf("pct %v want %v", r.CompletionPct, want)
	}
}

func TestDailyReport_AllDone(t *testing.T) {
	tr := &mockTasksRepo{tasks: []*tasks_domain.Task{{IsCompleted: true}, {IsCompleted: true}}}
	svc := NewReportsService(tr, &mockEventsRepo{})
	r, _ := svc.GetDailyReport(context.Background(), uuid.New(), time.Now())
	if r.CompletionPct != 100 {
		t.Fatalf("expected 100, got %v", r.CompletionPct)
	}
}

func TestDailyReport_TasksError(t *testing.T) {
	tr := &mockTasksRepo{err: errors.New("db down")}
	svc := NewReportsService(tr, &mockEventsRepo{})
	if _, err := svc.GetDailyReport(context.Background(), uuid.New(), time.Now()); err == nil {
		t.Fatalf("expected tasks error")
	}
}

func TestDailyReport_EventsError(t *testing.T) {
	er := &mockEventsRepo{err: errors.New("db down")}
	svc := NewReportsService(&mockTasksRepo{}, er)
	if _, err := svc.GetDailyReport(context.Background(), uuid.New(), time.Now()); err == nil {
		t.Fatalf("expected events error")
	}
}

func TestPeriodReport_Days(t *testing.T) {
	svc := NewReportsService(&mockTasksRepo{}, &mockEventsRepo{})
	from := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 9, 3, 0, 0, 0, 0, time.UTC)
	reps, err := svc.GetPeriodReport(context.Background(), uuid.New(), from, to)
	if err != nil {
		t.Fatal(err)
	}
	if len(reps) != 3 {
		t.Fatalf("expected 3 daily reports, got %d", len(reps))
	}
}

func TestPeriodReport_SingleDay(t *testing.T) {
	svc := NewReportsService(&mockTasksRepo{}, &mockEventsRepo{})
	d := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
	reps, err := svc.GetPeriodReport(context.Background(), uuid.New(), d, d)
	if err != nil {
		t.Fatal(err)
	}
	if len(reps) != 1 {
		t.Fatalf("expected 1, got %d", len(reps))
	}
}

func TestPeriodReport_InvertedRange_Empty(t *testing.T) {
	svc := NewReportsService(&mockTasksRepo{}, &mockEventsRepo{})
	from := time.Date(2026, 9, 5, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
	reps, err := svc.GetPeriodReport(context.Background(), uuid.New(), from, to)
	if err != nil {
		t.Fatal(err)
	}
	if len(reps) != 0 {
		t.Fatalf("inverted range should give 0 reports, got %d", len(reps))
	}
}

func TestDailyReport_JSONSnakeCase(t *testing.T) {
	// regression: DailyReport must serialize snake_case (F.10 contract)
	typ := reflect.TypeOf(struct {
		Date           string  `json:"date"`
		TotalTasks     int     `json:"total_tasks"`
		CompletedTasks int     `json:"completed_tasks"`
		CompletionPct  float64 `json:"completion_pct"`
		EventsCount    int     `json:"events_count"`
	}{})
	_ = typ
	// check actual DailyReport tags
	rt := reflect.TypeOf(struct {
		Date           interface{} `json:"date"`
		TotalTasks     interface{} `json:"total_tasks"`
		CompletedTasks interface{} `json:"completed_tasks"`
		CompletionPct  interface{} `json:"completion_pct"`
		EventsCount    interface{} `json:"events_count"`
	}{})
	_ = rt
	got := map[string]bool{}
	rrt := reflect.TypeOf(*mustDaily())
	for i := 0; i < rrt.NumField(); i++ {
		got[rrt.Field(i).Tag.Get("json")] = true
	}
	for _, want := range []string{"date", "total_tasks", "completed_tasks", "completion_pct", "events_count"} {
		if !got[want] {
			t.Fatalf("missing json tag %q, got %v", want, got)
		}
	}
}
