package postgres

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
