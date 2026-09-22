package service

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

type RemindersServiceInterface interface {
	ScheduleForEvent(ctx context.Context, event *domain.Event, userID uuid.UUID) error
	UpdateRemindersForEvent(ctx context.Context, event *domain.Event, userID uuid.UUID) error
	DeleteRemindersForEvent(ctx context.Context, eventID uuid.UUID) error
}

type EventsService interface {
	CreateEvent(ctx context.Context, userID uuid.UUID, input domain.CreateEventInput) (*domain.EventResponse, error)
	GetEvent(ctx context.Context, userID uuid.UUID, eventID uuid.UUID) (*domain.EventResponse, error)
	ListEvents(ctx context.Context, userID uuid.UUID, filter domain.EventFilter) ([]*domain.EventResponse, error)
	UpdateEvent(ctx context.Context, userID uuid.UUID, eventID uuid.UUID, input domain.UpdateEventInput) (*domain.EventResponse, error)
	DeleteEvent(ctx context.Context, userID uuid.UUID, eventID uuid.UUID) error
}
