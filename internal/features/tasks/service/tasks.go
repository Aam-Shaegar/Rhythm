package service

import (
	"context"
	"time"

	core_errors "github.com/Aam-Shaegar/Rhythm/internal/core/errors"
	"github.com/Aam-Shaegar/Rhythm/internal/features/tasks/domain"
	"github.com/Aam-Shaegar/Rhythm/internal/features/tasks/repository/postgres"
	"github.com/google/uuid"
)

type TasksServiceImpl struct {
	repo          postgres.TasksRepository
	remindersRepo RemindersServiceInterface
}

func NewTasksService(repo postgres.TasksRepository, remindersRepo RemindersServiceInterface) *TasksServiceImpl {
	return &TasksServiceImpl{
		repo:          repo,
		remindersRepo: remindersRepo,
	}
}

func (s *TasksServiceImpl) CreateTask(ctx context.Context, userID uuid.UUID, input domain.CreateTaskInput) (*domain.TaskResponse, error) {
	dueAt, err := time.Parse(time.RFC3339, input.DueAt)
	if err != nil {
		return nil, core_errors.ErrInvalidArgument
	}

	var recurrenceEnd *time.Time
	if input.RecurrenceEnd != nil {
		t, err := time.Parse(time.RFC3339, *input.RecurrenceEnd)
		if err != nil {
			return nil, core_errors.ErrInvalidArgument
		}
		recurrenceEnd = &t
	}

	now := time.Now().UTC()
	task := &domain.Task{
		ID:             uuid.New(),
		UserID:         userID,
		Title:          input.Title,
		Description:    input.Description,
		DueAt:          dueAt,
		IsCompleted:    false,
		RecurrenceType: input.RecurrenceType,
		RecurrenceEnd:  recurrenceEnd,
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	if err := s.repo.Create(ctx, task); err != nil {
		return nil, err
	}

	// Generate recurring tasks if this is a parent task
	if input.RecurrenceType != nil && recurrenceEnd != nil {
		if err := s.generateRecurringForTask(ctx, task); err != nil {
			// Log error but don't fail the creation
			// In production, you'd want to handle this more gracefully
		}
	}

	return s.toResponse(task), nil
}

func (s *TasksServiceImpl) GetTask(ctx context.Context, userID uuid.UUID, taskID uuid.UUID) (*domain.TaskResponse, error) {
	task, err := s.repo.GetByID(ctx, taskID)
	if err != nil {
		return nil, err
	}

	if task.UserID != userID {
		return nil, core_errors.ErrNotFound
	}

	return s.toResponse(task), nil
}

func (s *TasksServiceImpl) ListTasks(ctx context.Context, userID uuid.UUID, filter domain.TaskFilter) ([]*domain.TaskResponse, error) {
	filter.UserID = userID

	tasks, err := s.repo.GetByUserID(ctx, filter)
	if err != nil {
		return nil, err
	}

	responses := make([]*domain.TaskResponse, len(tasks))
	for i, t := range tasks {
		responses[i] = s.toResponse(t)
	}
	return responses, nil
}

func (s *TasksServiceImpl) UpdateTask(ctx context.Context, userID uuid.UUID, taskID uuid.UUID, input domain.UpdateTaskInput) (*domain.TaskResponse, error) {
	task, err := s.repo.GetByID(ctx, taskID)
	if err != nil {
		return nil, err
	}

	if task.UserID != userID {
		return nil, core_errors.ErrNotFound
	}

	if input.Title != nil {
		task.Title = *input.Title
	}
	if input.Description != nil {
		task.Description = input.Description
	}
	if input.DueAt != nil {
		dueAt, err := time.Parse(time.RFC3339, *input.DueAt)
		if err != nil {
			return nil, core_errors.ErrInvalidArgument
		}
		task.DueAt = dueAt
	}
	if input.RecurrenceType != nil {
		task.RecurrenceType = input.RecurrenceType
	}
	if input.RecurrenceEnd != nil {
		t, err := time.Parse(time.RFC3339, *input.RecurrenceEnd)
		if err != nil {
			return nil, core_errors.ErrInvalidArgument
		}
		task.RecurrenceEnd = &t
	}
	if input.IsCompleted != nil {
		task.IsCompleted = *input.IsCompleted
		if task.IsCompleted {
			now := time.Now().UTC()
			task.CompletedAt = &now
		} else {
			task.CompletedAt = nil
		}
	}

	if err := s.repo.Update(ctx, task); err != nil {
		return nil, err
	}

	return s.toResponse(task), nil
}

func (s *TasksServiceImpl) CompleteTask(ctx context.Context, userID uuid.UUID, taskID uuid.UUID) (*domain.TaskResponse, error) {
	task, err := s.repo.GetByID(ctx, taskID)
	if err != nil {
		return nil, err
	}

	if task.UserID != userID {
		return nil, core_errors.ErrNotFound
	}

	if task.IsCompleted {
		return s.toResponse(task), nil
	}

	now := time.Now().UTC()
	task.IsCompleted = true
	task.CompletedAt = &now

	if err := s.repo.Update(ctx, task); err != nil {
		return nil, err
	}

	return s.toResponse(task), nil
}

func (s *TasksServiceImpl) DeleteTask(ctx context.Context, userID uuid.UUID, taskID uuid.UUID) error {
	task, err := s.repo.GetByID(ctx, taskID)
	if err != nil {
		return err
	}

	if task.UserID != userID {
		return core_errors.ErrNotFound
	}

	return s.repo.Delete(ctx, taskID)
}

func (s *TasksServiceImpl) GenerateRecurringTasks(ctx context.Context, before time.Time) error {
	// Get all users with recurring parent tasks
	// This is a simplified version - in production you'd want to paginate users
	users, err := s.getUsersWithRecurringTasks(ctx)
	if err != nil {
		return err
	}

	for _, userID := range users {
		parents, err := s.repo.GetRecurringParents(ctx, userID, before)
		if err != nil {
			return err
		}

		for _, parent := range parents {
			if err := s.generateRecurringForTask(ctx, parent); err != nil {
				return err
			}
		}
	}

	return nil
}

func (s *TasksServiceImpl) getUsersWithRecurringTasks(ctx context.Context) ([]uuid.UUID, error) {
	return s.repo.GetUsersWithRecurringTasks(ctx, time.Now().UTC())
}

func (s *TasksServiceImpl) generateRecurringForTask(ctx context.Context, parent *domain.Task) error {
	if parent.RecurrenceType == nil || parent.RecurrenceEnd == nil {
		return nil
	}

	// Get existing children to find the latest due date
	children, err := s.repo.GetByParentID(ctx, parent.ID)
	if err != nil {
		return err
	}

	var lastDue time.Time
	if len(children) > 0 {
		lastDue = children[len(children)-1].DueAt
	} else {
		lastDue = parent.DueAt
	}

	// Generate next occurrences
	nextDue := s.nextOccurrence(lastDue, *parent.RecurrenceType)
	for nextDue.Before(*parent.RecurrenceEnd) || nextDue.Equal(*parent.RecurrenceEnd) {
		child := &domain.Task{
			ID:             uuid.New(),
			UserID:         parent.UserID,
			Title:          parent.Title,
			Description:    parent.Description,
			DueAt:          nextDue,
			IsCompleted:    false,
			RecurrenceType: nil, // Children don't recurse
			RecurrenceEnd:  nil,
			ParentTaskID:   &parent.ID,
			CreatedAt:      time.Now().UTC(),
			UpdatedAt:      time.Now().UTC(),
		}

		if err := s.repo.Create(ctx, child); err != nil {
			return err
		}

		// Create reminder for this task
		if err := s.createTaskReminders(ctx, child); err != nil {
			// Log but continue
		}

		nextDue = s.nextOccurrence(nextDue, *parent.RecurrenceType)
	}

	return nil
}

func (s *TasksServiceImpl) nextOccurrence(current time.Time, rtype domain.RecurrenceType) time.Time {
	switch rtype {
	case domain.RecurrenceDaily:
		return current.Add(24 * time.Hour)
	case domain.RecurrenceWeekly:
		return current.Add(7 * 24 * time.Hour)
	case domain.RecurrenceMonthly:
		return current.AddDate(0, 1, 0)
	case domain.RecurrenceYearly:
		return current.AddDate(1, 0, 0)
	default:
		return current.Add(24 * time.Hour)
	}
}

func (s *TasksServiceImpl) createTaskReminders(ctx context.Context, task *domain.Task) error {
	return s.remindersRepo.ScheduleForTask(ctx, task, task.UserID)
}

func (s *TasksServiceImpl) toResponse(task *domain.Task) *domain.TaskResponse {
	return &domain.TaskResponse{
		ID:              task.ID,
		Title:           task.Title,
		Description:     task.Description,
		DueAt:           task.DueAt,
		IsCompleted:     task.IsCompleted,
		CompletedAt:     task.CompletedAt,
		RecurrenceType:  task.RecurrenceType,
		RecurrenceEnd:   task.RecurrenceEnd,
		ParentTaskID:    task.ParentTaskID,
		CreatedAt:       task.CreatedAt,
		UpdatedAt:       task.UpdatedAt,
	}
}