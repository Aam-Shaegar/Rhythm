package postgres

import (
	"context"
	"time"

	"github.com/Aam-Shaegar/Rhythm/internal/features/events/domain"
	"github.com/google/uuid"
)

type EventsRepository interface {
	Create(ctx context.Context, event *domain.Event) error
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Event, error)
	GetByUserID(ctx context.Context, filter domain.EventFilter) ([]*domain.Event, error)
	Update(ctx context.Context, event *domain.Event) error
	Delete(ctx context.Context, id uuid.UUID) error
	GetUpcoming(ctx context.Context, userID uuid.UUID, before time.Time, limit int) ([]*domain.Event, error)
}
