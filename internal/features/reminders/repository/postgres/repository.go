package postgres

import (
	"context"
	"time"

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