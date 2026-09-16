package service

import (
	"context"
	"errors"
	"testing"
	"time"

	core_errors "github.com/Aam-Shaegar/Rhythm/internal/core/errors"
	"github.com/Aam-Shaegar/Rhythm/internal/features/tasks/domain"
	"github.com/google/uuid"
)

type mockTasksRepo struct {
	tasks map[uuid.UUID]*domain.Task
	byUser map[uuid.UUID][]*domain.Task
	byParent map[uuid.UUID][]*domain.Task
}

func newMockTasksRepo() *mockTasksRepo {
	return &mockTasksRepo{
		tasks: make(map[uuid.UUID]*domain.Task),
		byUser: make(map[uuid.UUID][]*domain.Task),
		byParent: make(map[uuid.UUID][]*domain.Task),
	}
}

func (m *mockTasksRepo) Create(ctx context.Context, task *domain.Task) error {
	if _, exists := m.tasks[task.ID]; exists {
		return core_errors.ErrConflict
	}
	m.tasks[task.ID] = task
	m.byUser[task.UserID] = append(m.byUser[task.UserID], task)
	if task.ParentTaskID != nil {
		m.byParent[*task.ParentTaskID] = append(m.byParent[*task.ParentTaskID], task)
	}
	return nil
}

func (m *mockTasksRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.Task, error) {
	if t, ok := m.tasks[id]; ok {
		return t, nil
	}
	return nil, core_errors.ErrNotFound
}

func (m *mockTasksRepo) GetByUserID(ctx context.Context, filter domain.TaskFilter) ([]*domain.Task, error) {
	userTasks, ok := m.byUser[filter.UserID]
	if !ok {
		return []*domain.Task{}, nil
	}

	var result []*domain.Task
	for _, t := range userTasks {
		if filter.DateFrom != nil && t.DueAt.Before(*filter.DateFrom) {
			continue
		}
		if filter.DateTo != nil && t.DueAt.After(*filter.DateTo) {
			continue
		}
		if filter.Completed != nil && t.IsCompleted != *filter.Completed {
			continue
		}
		if filter.Recurring != nil {
			isRecurring := t.RecurrenceType != nil
			if isRecurring != *filter.Recurring {
				continue
			}
		}
		result = append(result, t)
	}
	return result, nil
}

func (m *mockTasksRepo) GetByParentID(ctx context.Context, parentID uuid.UUID) ([]*domain.Task, error) {
	if tasks, ok := m.byParent[parentID]; ok {
		return tasks, nil
	}
	return []*domain.Task{}, nil
}

func (m *mockTasksRepo) GetRecurringParents(ctx context.Context, userID uuid.UUID, before time.Time) ([]*domain.Task, error) {
	userTasks, ok := m.byUser[userID]
	if !ok {
		return []*domain.Task{}, nil
	}

	var result []*domain.Task
	for _, t := range userTasks {
		if t.RecurrenceType != nil && t.ParentTaskID == nil {
			if t.RecurrenceEnd == nil || t.RecurrenceEnd.After(before) || t.RecurrenceEnd.Equal(before) {
				result = append(result, t)
			}
		}
	}
	return result, nil
}

func (m *mockTasksRepo) GetUsersWithRecurringTasks(ctx context.Context, before time.Time) ([]uuid.UUID, error) {
	var result []uuid.UUID
	seen := make(map[uuid.UUID]bool)
	for _, t := range m.tasks {
		if t.RecurrenceType != nil && t.ParentTaskID == nil {
			if t.RecurrenceEnd == nil || t.RecurrenceEnd.After(before) || t.RecurrenceEnd.Equal(before) {
				if !seen[t.UserID] {
					seen[t.UserID] = true
					result = append(result, t.UserID)
				}
			}
		}
	}
	return result, nil
}

func (m *mockTasksRepo) Update(ctx context.Context, task *domain.Task) error {
	if _, ok := m.tasks[task.ID]; !ok {
		return core_errors.ErrNotFound
	}
	m.tasks[task.ID] = task

	// Update in byUser
	for i, t := range m.byUser[task.UserID] {
		if t.ID == task.ID {
			m.byUser[task.UserID][i] = task
			break
		}
	}
	// Update in byParent if needed
	if task.ParentTaskID != nil {
		for i, t := range m.byParent[*task.ParentTaskID] {
			if t.ID == task.ID {
				m.byParent[*task.ParentTaskID][i] = task
				break
			}
		}
	}
	return nil
}

func (m *mockTasksRepo) Delete(ctx context.Context, id uuid.UUID) error {
	task, ok := m.tasks[id]
	if !ok {
		return core_errors.ErrNotFound
	}
	delete(m.tasks, id)

	// Remove from byUser
	userTasks := m.byUser[task.UserID]
	for i, t := range userTasks {
		if t.ID == id {
			m.byUser[task.UserID] = append(userTasks[:i], userTasks[i+1:]...)
			break
		}
	}
	// Remove from byParent
	if task.ParentTaskID != nil {
		parentTasks := m.byParent[*task.ParentTaskID]
		for i, t := range parentTasks {
			if t.ID == id {
				m.byParent[*task.ParentTaskID] = append(parentTasks[:i], parentTasks[i+1:]...)
				break
			}
		}
	}
	return nil
}

type mockRemindersRepo struct {
	reminders []*domain.Reminder
}

func newMockRemindersRepo() *mockRemindersRepo {
	return &mockRemindersRepo{
		reminders: make([]*domain.Reminder, 0),
	}
}

func (m *mockRemindersRepo) CreateReminders(ctx context.Context, reminders []*domain.Reminder) error {
	m.reminders = append(m.reminders, reminders...)
	return nil
}

func (m *mockRemindersRepo) ScheduleForTask(ctx context.Context, task *domain.Task, userID uuid.UUID) error {
	// Mock implementation - just track that it was called
	return nil
}

func (m *mockRemindersRepo) UpdateRemindersForTask(ctx context.Context, task *domain.Task, userID uuid.UUID) error {
	return nil
}

func (m *mockRemindersRepo) DeleteRemindersForTask(ctx context.Context, taskID uuid.UUID) error {
	return nil
}

func TestCreateTask_Success(t *testing.T) {
	tasksRepo := newMockTasksRepo()
	remindersRepo := newMockRemindersRepo()
	svc := NewTasksService(tasksRepo, remindersRepo)

	userID := uuid.New()
	input := domain.CreateTaskInput{
		Title:       "Test Task",
		Description: strPtr("Test Description"),
		DueAt:       time.Now().Add(time.Hour).Format(time.RFC3339),
	}

	task, err := svc.CreateTask(context.Background(), userID, input)
	if err != nil {
		t.Fatalf("CreateTask failed: %v", err)
	}

	if task.Title != "Test Task" {
		t.Errorf("expected title 'Test Task', got %s", task.Title)
	}
	if task.Description == nil || *task.Description != "Test Description" {
		t.Errorf("expected description 'Test Description', got %v", task.Description)
	}
	if task.IsCompleted {
		t.Error("expected task to not be completed")
	}
}

func TestCreateTask_WithRecurrence(t *testing.T) {
	tasksRepo := newMockTasksRepo()
	remindersRepo := newMockRemindersRepo()
	svc := NewTasksService(tasksRepo, remindersRepo)

	userID := uuid.New()
	now := time.Now()
	recurEnd := now.Add(7 * 24 * time.Hour)

	input := domain.CreateTaskInput{
		Title:          "Recurring Task",
		DueAt:          now.Add(time.Hour).Format(time.RFC3339),
		RecurrenceType: ptr(domain.RecurrenceDaily),
		RecurrenceEnd:  strPtr(recurEnd.Format(time.RFC3339)),
	}

	task, err := svc.CreateTask(context.Background(), userID, input)
	if err != nil {
		t.Fatalf("CreateTask failed: %v", err)
	}

	if task.RecurrenceType == nil {
		t.Fatal("expected recurrence type to be set")
	}
	if *task.RecurrenceType != domain.RecurrenceDaily {
		t.Errorf("expected daily recurrence, got %s", *task.RecurrenceType)
	}
}

func TestCreateTask_InvalidTimeFormat(t *testing.T) {
	tasksRepo := newMockTasksRepo()
	remindersRepo := newMockRemindersRepo()
	svc := NewTasksService(tasksRepo, remindersRepo)

	userID := uuid.New()
	input := domain.CreateTaskInput{
		Title: "Test Task",
		DueAt: "invalid-time",
	}

	_, err := svc.CreateTask(context.Background(), userID, input)
	if err == nil {
		t.Fatal("expected error for invalid time format")
	}
	if !errors.Is(err, core_errors.ErrInvalidArgument) {
		t.Errorf("expected ErrInvalidArgument, got %v", err)
	}
}

func TestGetTask_Success(t *testing.T) {
	tasksRepo := newMockTasksRepo()
	remindersRepo := newMockRemindersRepo()
	svc := NewTasksService(tasksRepo, remindersRepo)

	userID := uuid.New()
	taskID := uuid.New()
	task := &domain.Task{
		ID:          taskID,
		UserID:      userID,
		Title:       "Test Task",
		Description: strPtr("Test Description"),
		DueAt:       time.Now().Add(time.Hour),
		IsCompleted: false,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	tasksRepo.Create(context.Background(), task)

	result, err := svc.GetTask(context.Background(), userID, taskID)
	if err != nil {
		t.Fatalf("GetTask failed: %v", err)
	}

	if result.ID != taskID {
		t.Errorf("expected task ID %v, got %v", taskID, result.ID)
	}
	if result.Title != "Test Task" {
		t.Errorf("expected title 'Test Task', got %s", result.Title)
	}
}

func TestGetTask_NotFound(t *testing.T) {
	tasksRepo := newMockTasksRepo()
	remindersRepo := newMockRemindersRepo()
	svc := NewTasksService(tasksRepo, remindersRepo)

	_, err := svc.GetTask(context.Background(), uuid.New(), uuid.New())
	if err == nil {
		t.Fatal("expected not found error")
	}
	if !errors.Is(err, core_errors.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestGetTask_WrongUser(t *testing.T) {
	tasksRepo := newMockTasksRepo()
	remindersRepo := newMockRemindersRepo()
	svc := NewTasksService(tasksRepo, remindersRepo)

	userID := uuid.New()
	otherUserID := uuid.New()
	taskID := uuid.New()
	task := &domain.Task{
		ID:          taskID,
		UserID:      userID,
		Title:       "Test Task",
		DueAt:       time.Now().Add(time.Hour),
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	tasksRepo.Create(context.Background(), task)

	_, err := svc.GetTask(context.Background(), otherUserID, taskID)
	if err == nil {
		t.Fatal("expected not found error for other user")
	}
	if !errors.Is(err, core_errors.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestListTasks_Success(t *testing.T) {
	tasksRepo := newMockTasksRepo()
	remindersRepo := newMockRemindersRepo()
	svc := NewTasksService(tasksRepo, remindersRepo)

	userID := uuid.New()
	now := time.Now()

	task1 := &domain.Task{
		ID:          uuid.New(),
		UserID:      userID,
		Title:       "Task 1",
		DueAt:       now.Add(time.Hour),
		IsCompleted: false,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	task2 := &domain.Task{
		ID:          uuid.New(),
		UserID:      userID,
		Title:       "Task 2",
		DueAt:       now.Add(2 * time.Hour),
		IsCompleted: false,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	tasksRepo.Create(context.Background(), task1)
	tasksRepo.Create(context.Background(), task2)

	filter := domain.TaskFilter{}
	tasks, err := svc.ListTasks(context.Background(), userID, filter)
	if err != nil {
		t.Fatalf("ListTasks failed: %v", err)
	}

	if len(tasks) != 2 {
		t.Errorf("expected 2 tasks, got %d", len(tasks))
	}
}

func TestListTasks_WithDateFilter(t *testing.T) {
	tasksRepo := newMockTasksRepo()
	remindersRepo := newMockRemindersRepo()
	svc := NewTasksService(tasksRepo, remindersRepo)

	userID := uuid.New()
	now := time.Now()

	task1 := &domain.Task{
		ID:          uuid.New(),
		UserID:      userID,
		Title:       "Task 1",
		DueAt:       now.Add(time.Hour),
		IsCompleted: false,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	task2 := &domain.Task{
		ID:          uuid.New(),
		UserID:      userID,
		Title:       "Task 2",
		DueAt:       now.Add(24 * time.Hour),
		IsCompleted: false,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	tasksRepo.Create(context.Background(), task1)
	tasksRepo.Create(context.Background(), task2)

	dateFrom := now.Add(12 * time.Hour)
	filter := domain.TaskFilter{
		DateFrom: &dateFrom,
	}
	tasks, err := svc.ListTasks(context.Background(), userID, filter)
	if err != nil {
		t.Fatalf("ListTasks failed: %v", err)
	}

	if len(tasks) != 1 {
		t.Errorf("expected 1 task after date filter, got %d", len(tasks))
	}
	if tasks[0].Title != "Task 2" {
		t.Errorf("expected 'Task 2', got %s", tasks[0].Title)
	}
}

func TestListTasks_CompletedFilter(t *testing.T) {
	tasksRepo := newMockTasksRepo()
	remindersRepo := newMockRemindersRepo()
	svc := NewTasksService(tasksRepo, remindersRepo)

	userID := uuid.New()
	now := time.Now()

	task1 := &domain.Task{
		ID:          uuid.New(),
		UserID:      userID,
		Title:       "Task 1",
		DueAt:       now.Add(time.Hour),
		IsCompleted: true,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	task2 := &domain.Task{
		ID:          uuid.New(),
		UserID:      userID,
		Title:       "Task 2",
		DueAt:       now.Add(2 * time.Hour),
		IsCompleted: false,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	tasksRepo.Create(context.Background(), task1)
	tasksRepo.Create(context.Background(), task2)

	completed := true
	filter := domain.TaskFilter{
		Completed: &completed,
	}
	tasks, err := svc.ListTasks(context.Background(), userID, filter)
	if err != nil {
		t.Fatalf("ListTasks failed: %v", err)
	}

	if len(tasks) != 1 {
		t.Errorf("expected 1 completed task, got %d", len(tasks))
	}
	if tasks[0].Title != "Task 1" {
		t.Errorf("expected 'Task 1', got %s", tasks[0].Title)
	}
}

func TestUpdateTask_Success(t *testing.T) {
	tasksRepo := newMockTasksRepo()
	remindersRepo := newMockRemindersRepo()
	svc := NewTasksService(tasksRepo, remindersRepo)

	userID := uuid.New()
	taskID := uuid.New()
	task := &domain.Task{
		ID:          taskID,
		UserID:      userID,
		Title:       "Original Title",
		Description: strPtr("Original Description"),
		DueAt:       time.Now().Add(time.Hour),
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	tasksRepo.Create(context.Background(), task)

	newTitle := "Updated Title"
	newDesc := "Updated Description"
	newDue := time.Now().Add(3 * time.Hour).Format(time.RFC3339)

	input := domain.UpdateTaskInput{
		Title:       &newTitle,
		Description: &newDesc,
		DueAt:       &newDue,
	}

	result, err := svc.UpdateTask(context.Background(), userID, taskID, input)
	if err != nil {
		t.Fatalf("UpdateTask failed: %v", err)
	}

	if result.Title != "Updated Title" {
		t.Errorf("expected title 'Updated Title', got %s", result.Title)
	}
	if result.Description == nil || *result.Description != "Updated Description" {
		t.Errorf("expected description 'Updated Description', got %v", result.Description)
	}
}

func TestUpdateTask_WrongUser(t *testing.T) {
	tasksRepo := newMockTasksRepo()
	remindersRepo := newMockRemindersRepo()
	svc := NewTasksService(tasksRepo, remindersRepo)

	userID := uuid.New()
	otherUserID := uuid.New()
	taskID := uuid.New()
	task := &domain.Task{
		ID:          taskID,
		UserID:      userID,
		Title:       "Test Task",
		DueAt:       time.Now().Add(time.Hour),
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	tasksRepo.Create(context.Background(), task)

	input := domain.UpdateTaskInput{
		Title: strPtr("New Title"),
	}

	_, err := svc.UpdateTask(context.Background(), otherUserID, taskID, input)
	if err == nil {
		t.Fatal("expected not found error for other user")
	}
	if !errors.Is(err, core_errors.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestCompleteTask_Success(t *testing.T) {
	tasksRepo := newMockTasksRepo()
	remindersRepo := newMockRemindersRepo()
	svc := NewTasksService(tasksRepo, remindersRepo)

	userID := uuid.New()
	taskID := uuid.New()
	task := &domain.Task{
		ID:          taskID,
		UserID:      userID,
		Title:       "Test Task",
		DueAt:       time.Now().Add(time.Hour),
		IsCompleted: false,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	tasksRepo.Create(context.Background(), task)

	result, err := svc.CompleteTask(context.Background(), userID, taskID)
	if err != nil {
		t.Fatalf("CompleteTask failed: %v", err)
	}

	if !result.IsCompleted {
		t.Error("expected task to be completed")
	}
	if result.CompletedAt == nil {
		t.Error("expected completed_at to be set")
	}
}

func TestCompleteTask_AlreadyCompleted(t *testing.T) {
	tasksRepo := newMockTasksRepo()
	remindersRepo := newMockRemindersRepo()
	svc := NewTasksService(tasksRepo, remindersRepo)

	userID := uuid.New()
	taskID := uuid.New()
	task := &domain.Task{
		ID:          taskID,
		UserID:      userID,
		Title:       "Test Task",
		DueAt:       time.Now().Add(time.Hour),
		IsCompleted: true,
		CompletedAt: ptr(time.Now()),
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	tasksRepo.Create(context.Background(), task)

	result, err := svc.CompleteTask(context.Background(), userID, taskID)
	if err != nil {
		t.Fatalf("CompleteTask failed: %v", err)
	}

	if !result.IsCompleted {
		t.Error("expected task to remain completed")
	}
}

func TestDeleteTask_Success(t *testing.T) {
	tasksRepo := newMockTasksRepo()
	remindersRepo := newMockRemindersRepo()
	svc := NewTasksService(tasksRepo, remindersRepo)

	userID := uuid.New()
	taskID := uuid.New()
	task := &domain.Task{
		ID:          taskID,
		UserID:      userID,
		Title:       "Test Task",
		DueAt:       time.Now().Add(time.Hour),
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	tasksRepo.Create(context.Background(), task)

	err := svc.DeleteTask(context.Background(), userID, taskID)
	if err != nil {
		t.Fatalf("DeleteTask failed: %v", err)
	}

	_, err = svc.GetTask(context.Background(), userID, taskID)
	if err == nil {
		t.Fatal("expected not found after delete")
	}
	if !errors.Is(err, core_errors.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestDeleteTask_WrongUser(t *testing.T) {
	tasksRepo := newMockTasksRepo()
	remindersRepo := newMockRemindersRepo()
	svc := NewTasksService(tasksRepo, remindersRepo)

	userID := uuid.New()
	otherUserID := uuid.New()
	taskID := uuid.New()
	task := &domain.Task{
		ID:          taskID,
		UserID:      userID,
		Title:       "Test Task",
		DueAt:       time.Now().Add(time.Hour),
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	tasksRepo.Create(context.Background(), task)

	err := svc.DeleteTask(context.Background(), otherUserID, taskID)
	if err == nil {
		t.Fatal("expected not found error for other user")
	}
	if !errors.Is(err, core_errors.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func ptr[T any](v T) *T {
	return &v
}

func strPtr(s string) *string {
	return &s
}