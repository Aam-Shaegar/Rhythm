package service

import (
	"context"
	"time"

	"go.uber.org/zap"

	events_domain "github.com/Aam-Shaegar/Rhythm/internal/features/events/domain"
	tasks_domain "github.com/Aam-Shaegar/Rhythm/internal/features/tasks/domain"
	"github.com/Aam-Shaegar/Rhythm/internal/features/reminders/domain"
	"github.com/google/uuid"
)

type RemindersRepository interface {
	CreateReminders(ctx context.Context, reminders []*domain.Reminder) error
	GetPending(ctx context.Context, before time.Time, limit int) ([]*domain.Reminder, error)
	MarkSent(ctx context.Context, ids []uuid.UUID) error
	GetByEntity(ctx context.Context, entityType string, entityID uuid.UUID) ([]*domain.Reminder, error)
	DeleteByEntity(ctx context.Context, entityType string, entityID uuid.UUID) error
}

type Logger interface {
	Debug(msg string, fields ...zap.Field)
	Info(msg string, fields ...zap.Field)
	Warn(msg string, fields ...zap.Field)
	Error(msg string, fields ...zap.Field)
}

type RemindersService interface {
	ScheduleForEvent(ctx context.Context, event *events_domain.Event, userID uuid.UUID) error
	ScheduleForTask(ctx context.Context, task *tasks_domain.Task, userID uuid.UUID) error
	UpdateRemindersForEvent(ctx context.Context, event *events_domain.Event, userID uuid.UUID) error
	UpdateRemindersForTask(ctx context.Context, task *tasks_domain.Task, userID uuid.UUID) error
	DeleteRemindersForEvent(ctx context.Context, eventID uuid.UUID) error
	DeleteRemindersForTask(ctx context.Context, taskID uuid.UUID) error
	ProcessPendingReminders(ctx context.Context, before time.Time, limit int) (int, error)
	StartWorker(ctx context.Context, interval time.Duration, logger Logger)
}