package service

import (
	"context"
	"time"

	core_errors "github.com/Aam-Shaegar/Rhythm/internal/core/errors"
	"github.com/Aam-Shaegar/Rhythm/internal/features/events/domain"
	"github.com/Aam-Shaegar/Rhythm/internal/features/events/repository/postgres"
	"github.com/google/uuid"
)

type EventsServiceImpl struct {
	repo       postgres.EventsRepository
	reminders  RemindersServiceInterface
}

func NewEventsService(repo postgres.EventsRepository, reminders RemindersServiceInterface) *EventsServiceImpl {
	return &EventsServiceImpl{
		repo:      repo,
		reminders: reminders,
	}
}

func (s *EventsServiceImpl) CreateEvent(ctx context.Context, userID uuid.UUID, input domain.CreateEventInput) (*domain.EventResponse, error) {
	startAt, err := time.Parse(time.RFC3339, input.StartAt)
	if err != nil {
		return nil, core_errors.ErrInvalidArgument
	}

	endAt, err := time.Parse(time.RFC3339, input.EndAt)
	if err != nil {
		return nil, core_errors.ErrInvalidArgument
	}

	if endAt.Before(startAt) || endAt.Equal(startAt) {
		return nil, core_errors.ErrInvalidArgument
	}

	now := time.Now().UTC()
	event := &domain.Event{
		ID:          uuid.New(),
		UserID:      userID,
		Title:       input.Title,
		Description: input.Description,
		StartAt:     startAt,
		EndAt:       endAt,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if err := s.repo.Create(ctx, event); err != nil {
		return nil, err
	}

	// Schedule reminders
	if err := s.reminders.ScheduleForEvent(ctx, event, userID); err != nil {
		// Log error but don't fail the creation
	}

	return s.toResponse(event), nil
}

func (s *EventsServiceImpl) GetEvent(ctx context.Context, userID uuid.UUID, eventID uuid.UUID) (*domain.EventResponse, error) {
	event, err := s.repo.GetByID(ctx, eventID)
	if err != nil {
		return nil, err
	}

	if event.UserID != userID {
		return nil, core_errors.ErrNotFound
	}

	return s.toResponse(event), nil
}

func (s *EventsServiceImpl) ListEvents(ctx context.Context, userID uuid.UUID, filter domain.EventFilter) ([]*domain.EventResponse, error) {
	filter.UserID = userID

	events, err := s.repo.GetByUserID(ctx, filter)
	if err != nil {
		return nil, err
	}

	responses := make([]*domain.EventResponse, len(events))
	for i, e := range events {
		responses[i] = s.toResponse(e)
	}
	return responses, nil
}

func (s *EventsServiceImpl) UpdateEvent(ctx context.Context, userID uuid.UUID, eventID uuid.UUID, input domain.UpdateEventInput) (*domain.EventResponse, error) {
	event, err := s.repo.GetByID(ctx, eventID)
	if err != nil {
		return nil, err
	}

	if event.UserID != userID {
		return nil, core_errors.ErrNotFound
	}

	if input.Title != nil {
		event.Title = *input.Title
	}
	if input.Description != nil {
		event.Description = input.Description
	}
	if input.StartAt != nil {
		startAt, err := time.Parse(time.RFC3339, *input.StartAt)
		if err != nil {
			return nil, core_errors.ErrInvalidArgument
		}
		event.StartAt = startAt
	}
	if input.EndAt != nil {
		endAt, err := time.Parse(time.RFC3339, *input.EndAt)
		if err != nil {
			return nil, core_errors.ErrInvalidArgument
		}
		event.EndAt = endAt
	}

	if event.EndAt.Before(event.StartAt) || event.EndAt.Equal(event.StartAt) {
		return nil, core_errors.ErrInvalidArgument
	}

	if err := s.repo.Update(ctx, event); err != nil {
		return nil, err
	}

	// Update reminders
	if err := s.reminders.UpdateRemindersForEvent(ctx, event, userID); err != nil {
		// Log error but don't fail the update
	}

	return s.toResponse(event), nil
}

func (s *EventsServiceImpl) DeleteEvent(ctx context.Context, userID uuid.UUID, eventID uuid.UUID) error {
	event, err := s.repo.GetByID(ctx, eventID)
	if err != nil {
		return err
	}

	if event.UserID != userID {
		return core_errors.ErrNotFound
	}

	// Delete reminders
	if err := s.reminders.DeleteRemindersForEvent(ctx, eventID); err != nil {
		// Log error but don't fail the delete
	}

	return s.repo.Delete(ctx, eventID)
}

func (s *EventsServiceImpl) toResponse(event *domain.Event) *domain.EventResponse {
	return &domain.EventResponse{
		ID:          event.ID,
		Title:       event.Title,
		Description: event.Description,
		StartAt:     event.StartAt,
		EndAt:       event.EndAt,
		CreatedAt:   event.CreatedAt,
		UpdatedAt:   event.UpdatedAt,
	}
}