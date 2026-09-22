package service

import (
	"context"
	"time"

	"github.com/Aam-Shaegar/Rhythm/internal/features/tasks/domain"
	"github.com/google/uuid"
)

type TasksRepository interface {
	Create(ctx context.Context, task *domain.Task) error
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Task, error)
	GetByUserID(ctx context.Context, filter domain.TaskFilter) ([]*domain.Task, error)
	GetByParentID(ctx context.Context, parentID uuid.UUID) ([]*domain.Task, error)
	GetRecurringParents(ctx context.Context, userID uuid.UUID, before time.Time) ([]*domain.Task, error)
	GetUsersWithRecurringTasks(ctx context.Context, before time.Time) ([]uuid.UUID, error)
	Update(ctx context.Context, task *domain.Task) error
	Delete(ctx context.Context, id uuid.UUID) error
}

type RemindersServiceInterface interface {
	ScheduleForTask(ctx context.Context, task *domain.Task, userID uuid.UUID) error
	UpdateRemindersForTask(ctx context.Context, task *domain.Task, userID uuid.UUID) error
	DeleteRemindersForTask(ctx context.Context, taskID uuid.UUID) error
}

type TasksService interface {
	CreateTask(ctx context.Context, userID uuid.UUID, input domain.CreateTaskInput) (*domain.TaskResponse, error)
	GetTask(ctx context.Context, userID uuid.UUID, taskID uuid.UUID) (*domain.TaskResponse, error)
	ListTasks(ctx context.Context, userID uuid.UUID, filter domain.TaskFilter) ([]*domain.TaskResponse, error)
	UpdateTask(ctx context.Context, userID uuid.UUID, taskID uuid.UUID, input domain.UpdateTaskInput) (*domain.TaskResponse, error)
	CompleteTask(ctx context.Context, userID uuid.UUID, taskID uuid.UUID) (*domain.TaskResponse, error)
	DeleteTask(ctx context.Context, userID uuid.UUID, taskID uuid.UUID) error
	GenerateRecurringTasks(ctx context.Context, before time.Time) error
}
