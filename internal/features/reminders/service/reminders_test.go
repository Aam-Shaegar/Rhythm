package service

import (
	"context"
	"errors"
	"testing"
	"time"

	events_domain "github.com/Aam-Shaegar/Rhythm/internal/features/events/domain"
	"github.com/Aam-Shaegar/Rhythm/internal/features/reminders/domain"
	tasks_domain "github.com/Aam-Shaegar/Rhythm/internal/features/tasks/domain"
	"github.com/google/uuid"
)

type mockRemindersRepo struct {
	created [][]*domain.Reminder
	pending []*domain.Reminder
	marked  [][]uuid.UUID
	deleted [][2]string // entityType + id string (simplified)
	err     error
}

func (m *mockRemindersRepo) CreateReminders(ctx context.Context, rs []*domain.Reminder) error {
	if m.err != nil {
		return m.err
	}
	m.created = append(m.created, rs)
	return nil
}
func (m *mockRemindersRepo) GetPending(ctx context.Context, before time.Time, limit int) ([]*domain.Reminder, error) {
	if m.err != nil {
		return nil, m.err
	}
	if len(m.pending) > limit {
		return m.pending[:limit], nil
	}
	return m.pending, nil
}
func (m *mockRemindersRepo) MarkSent(ctx context.Context, ids []uuid.UUID) error {
	if m.err != nil {
		return m.err
	}
	m.marked = append(m.marked, ids)
	return nil
}
func (m *mockRemindersRepo) GetByEntity(ctx context.Context, t string, id uuid.UUID) ([]*domain.Reminder, error) {
	return nil, nil
}
func (m *mockRemindersRepo) DeleteByEntity(ctx context.Context, t string, id uuid.UUID) error {
	if m.err != nil {
		return m.err
	}
	m.deleted = append(m.deleted, [2]string{t, id.String()})
	return nil
}

func TestScheduleForEvent_FutureCreatesThree(t *testing.T) {
	repo := &mockRemindersRepo{}
	svc := NewRemindersService(repo)
	ev := &events_domain.Event{ID: uuid.New(), StartAt: time.Now().UTC().Add(48 * time.Hour)}
	if err := svc.ScheduleForEvent(context.Background(), ev, uuid.New()); err != nil {
		t.Fatal(err)
	}
	if len(repo.created) != 1 || len(repo.created[0]) != 3 {
		t.Fatalf("expected 3 reminders, got %v", repo.created)
	}
}

func TestScheduleForEvent_PastCreatesNone(t *testing.T) {
	repo := &mockRemindersRepo{}
	svc := NewRemindersService(repo)
	ev := &events_domain.Event{ID: uuid.New(), StartAt: time.Now().UTC().Add(-48 * time.Hour)}
	if err := svc.ScheduleForEvent(context.Background(), ev, uuid.New()); err != nil {
		t.Fatal(err)
	}
	if len(repo.created) != 0 {
		t.Fatalf("past event must not schedule, got %d batches", len(repo.created))
	}
}

func TestScheduleForEvent_SoonOnlyFuture(t *testing.T) {
	// event in 30m: -24h and -1h are past, only -15m is future
	repo := &mockRemindersRepo{}
	svc := NewRemindersService(repo)
	ev := &events_domain.Event{ID: uuid.New(), StartAt: time.Now().UTC().Add(30 * time.Minute)}
	if err := svc.ScheduleForEvent(context.Background(), ev, uuid.New()); err != nil {
		t.Fatal(err)
	}
	if len(repo.created) != 1 || len(repo.created[0]) != 1 {
		t.Fatalf("expected 1 reminder, got %v", repo.created)
	}
}

func TestScheduleForTask_FutureAndPast(t *testing.T) {
	repo := &mockRemindersRepo{}
	svc := NewRemindersService(repo)
	future := &tasks_domain.Task{ID: uuid.New(), DueAt: time.Now().UTC().Add(48 * time.Hour)}
	if err := svc.ScheduleForTask(context.Background(), future, uuid.New()); err != nil {
		t.Fatal(err)
	}
	if len(repo.created[0]) != 2 {
		t.Fatalf("future task needs 2 reminders, got %d", len(repo.created[0]))
	}
	repo2 := &mockRemindersRepo{}
	svc2 := NewRemindersService(repo2)
	past := &tasks_domain.Task{ID: uuid.New(), DueAt: time.Now().UTC().Add(-time.Hour)}
	if err := svc2.ScheduleForTask(context.Background(), past, uuid.New()); err != nil {
		t.Fatal(err)
	}
	if len(repo2.created) != 0 {
		t.Fatalf("past task must not schedule")
	}
}

func TestUpdateReminders_DeletesThenSchedules(t *testing.T) {
	repo := &mockRemindersRepo{}
	svc := NewRemindersService(repo)
	ev := &events_domain.Event{ID: uuid.New(), StartAt: time.Now().UTC().Add(48 * time.Hour)}
	if err := svc.UpdateRemindersForEvent(context.Background(), ev, uuid.New()); err != nil {
		t.Fatal(err)
	}
	if len(repo.deleted) != 1 || repo.deleted[0][0] != "event" {
		t.Fatalf("expected delete event reminders, got %v", repo.deleted)
	}
	if len(repo.created) != 1 {
		t.Fatalf("expected reschedule")
	}
	task := &tasks_domain.Task{ID: uuid.New(), DueAt: time.Now().UTC().Add(48 * time.Hour)}
	if err := svc.UpdateRemindersForTask(context.Background(), task, uuid.New()); err != nil {
		t.Fatal(err)
	}
	if len(repo.deleted) != 2 || repo.deleted[1][0] != "task" {
		t.Fatalf("expected delete task reminders, got %v", repo.deleted)
	}
}

func TestProcessPending_Empty(t *testing.T) {
	repo := &mockRemindersRepo{pending: nil}
	svc := NewRemindersService(repo)
	n, err := svc.ProcessPendingReminders(context.Background(), time.Now(), 100)
	if err != nil || n != 0 {
		t.Fatalf("empty should be 0,nil got %d,%v", n, err)
	}
	if len(repo.marked) != 0 {
		t.Fatalf("nothing to mark")
	}
}

func TestProcessPending_MarksSent(t *testing.T) {
	repo := &mockRemindersRepo{pending: []*domain.Reminder{{ID: uuid.New()}, {ID: uuid.New()}}}
	svc := NewRemindersService(repo)
	n, err := svc.ProcessPendingReminders(context.Background(), time.Now(), 100)
	if err != nil || n != 2 {
		t.Fatalf("expected 2, got %d,%v", n, err)
	}
	if len(repo.marked) != 1 || len(repo.marked[0]) != 2 {
		t.Fatalf("expected mark 2 ids, got %v", repo.marked)
	}
}

func TestProcessPending_RepoError(t *testing.T) {
	repo := &mockRemindersRepo{err: errors.New("db")}
	svc := NewRemindersService(repo)
	if _, err := svc.ProcessPendingReminders(context.Background(), time.Now(), 10); err == nil {
		t.Fatalf("expected error")
	}
}

func TestDeleteReminders(t *testing.T) {
	repo := &mockRemindersRepo{}
	svc := NewRemindersService(repo)
	id := uuid.New()
	if err := svc.DeleteRemindersForEvent(context.Background(), id); err != nil {
		t.Fatal(err)
	}
	if err := svc.DeleteRemindersForTask(context.Background(), id); err != nil {
		t.Fatal(err)
	}
	if len(repo.deleted) != 2 {
		t.Fatalf("expected 2 deletes")
	}
}
