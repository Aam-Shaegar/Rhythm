package service

import (
	"context"
	"testing"
	"time"

	"errors"
	core_errors "github.com/Aam-Shaegar/Rhythm/internal/core/errors"
	"github.com/Aam-Shaegar/Rhythm/internal/features/tasks/domain"
	"github.com/google/uuid"
)

func TestCreateTask_InvalidDueAt(t *testing.T) {
	svc := NewTasksService(newMockTasksRepo(), newMockRemindersRepo())
	for _, bad := range []string{"", "not-a-date", "2026-09-20", "2026/09/20 10:00", "123"} {
		_, err := svc.CreateTask(context.Background(), uuid.New(), domain.CreateTaskInput{Title: "t", DueAt: bad})
		if !errors.Is(err, core_errors.ErrInvalidArgument) {
			t.Fatalf("due_at %q: expected ErrInvalidArgument, got %v", bad, err)
		}
	}
}

func TestCreateTask_InvalidRecurrenceEnd(t *testing.T) {
	svc := NewTasksService(newMockTasksRepo(), newMockRemindersRepo())
	rt := domain.RecurrenceDaily
	bad := "bad-date"
	_, err := svc.CreateTask(context.Background(), uuid.New(), domain.CreateTaskInput{
		Title: "t", DueAt: time.Now().UTC().Format(time.RFC3339), RecurrenceType: &rt, RecurrenceEnd: &bad,
	})
	if !errors.Is(err, core_errors.ErrInvalidArgument) {
		t.Fatalf("expected invalid recurrence_end, got %v", err)
	}
}

func TestUpdateTask_InvalidDueAt(t *testing.T) {
	repo := newMockTasksRepo()
	svc := NewTasksService(repo, newMockRemindersRepo())
	uid := uuid.New()
	created, _ := svc.CreateTask(context.Background(), uid, domain.CreateTaskInput{Title: "t", DueAt: time.Now().UTC().Format(time.RFC3339)})
	id, _ := uuid.Parse(created.ID.String())
	bad := "nope"
	_, err := svc.UpdateTask(context.Background(), uid, id, domain.UpdateTaskInput{DueAt: &bad})
	if !errors.Is(err, core_errors.ErrInvalidArgument) {
		t.Fatalf("expected invalid due_at, got %v", err)
	}
}

func TestUpdateTask_WrongUser_NotFound(t *testing.T) {
	repo := newMockTasksRepo()
	svc := NewTasksService(repo, newMockRemindersRepo())
	uid := uuid.New()
	created, _ := svc.CreateTask(context.Background(), uid, domain.CreateTaskInput{Title: "t", DueAt: time.Now().UTC().Format(time.RFC3339)})
	id, _ := uuid.Parse(created.ID.String())
	newTitle := "hacked"
	_, err := svc.UpdateTask(context.Background(), uuid.New(), id, domain.UpdateTaskInput{Title: &newTitle})
	if !errors.Is(err, core_errors.ErrNotFound) {
		t.Fatalf("wrong user should be NotFound, got %v", err)
	}
}

func TestCompleteTask_WrongUser(t *testing.T) {
	repo := newMockTasksRepo()
	svc := NewTasksService(repo, newMockRemindersRepo())
	uid := uuid.New()
	created, _ := svc.CreateTask(context.Background(), uid, domain.CreateTaskInput{Title: "t", DueAt: time.Now().UTC().Format(time.RFC3339)})
	id, _ := uuid.Parse(created.ID.String())
	_, err := svc.CompleteTask(context.Background(), uuid.New(), id)
	if !errors.Is(err, core_errors.ErrNotFound) {
		t.Fatalf("expected NotFound, got %v", err)
	}
}

func TestDeleteTask_NotFound(t *testing.T) {
	svc := NewTasksService(newMockTasksRepo(), newMockRemindersRepo())
	if err := svc.DeleteTask(context.Background(), uuid.New(), uuid.New()); !errors.Is(err, core_errors.ErrNotFound) {
		t.Fatalf("expected NotFound, got %v", err)
	}
}

func TestListTasks_Empty(t *testing.T) {
	svc := NewTasksService(newMockTasksRepo(), newMockRemindersRepo())
	list, err := svc.ListTasks(context.Background(), uuid.New(), domain.TaskFilter{})
	if err != nil || len(list) != 0 {
		t.Fatalf("empty should be [], got %v,%v", list, err)
	}
}

func TestRecurrence_WeeklyMonthlyYearly(t *testing.T) {
	for _, tc := range []struct {
		rt    domain.RecurrenceType
		start time.Time
		want  time.Time
	}{
		{domain.RecurrenceDaily, time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC), time.Date(2026, 1, 2, 10, 0, 0, 0, time.UTC)},
		{domain.RecurrenceWeekly, time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC), time.Date(2026, 1, 8, 10, 0, 0, 0, time.UTC)},
		{domain.RecurrenceMonthly, time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC), time.Date(2026, 2, 15, 10, 0, 0, 0, time.UTC)},
		{domain.RecurrenceYearly, time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC), time.Date(2027, 1, 15, 10, 0, 0, 0, time.UTC)},
	} {
		svc := NewTasksService(newMockTasksRepo(), newMockRemindersRepo())
		got := svc.nextOccurrence(tc.start, tc.rt)
		if !got.Equal(tc.want) {
			t.Fatalf("%s: got %v want %v", tc.rt, got, tc.want)
		}
	}
}

func TestGenerateRecurring_EndBeforeStart_NoChildren(t *testing.T) {
	repo := newMockTasksRepo()
	svc := NewTasksService(repo, newMockRemindersRepo())
	uid := uuid.New()
	rt := domain.RecurrenceDaily
	due := time.Date(2026, 9, 20, 10, 0, 0, 0, time.UTC)
	end := time.Date(2026, 9, 19, 10, 0, 0, 0, time.UTC) // before due
	parent := &domain.Task{ID: uuid.New(), UserID: uid, Title: "p", DueAt: due, RecurrenceType: &rt, RecurrenceEnd: &end}
	_ = repo.Create(context.Background(), parent)
	if err := svc.generateRecurringForTask(context.Background(), parent); err != nil {
		t.Fatal(err)
	}
	children, _ := repo.GetByParentID(context.Background(), parent.ID)
	if len(children) != 0 {
		t.Fatalf("end before start should create 0 children, got %d", len(children))
	}
}

func TestGenerateRecurring_NoRecurrence_Noop(t *testing.T) {
	svc := NewTasksService(newMockTasksRepo(), newMockRemindersRepo())
	parent := &domain.Task{ID: uuid.New(), DueAt: time.Now()}
	if err := svc.generateRecurringForTask(context.Background(), parent); err != nil {
		t.Fatalf("nil recurrence should be noop, got %v", err)
	}
}
