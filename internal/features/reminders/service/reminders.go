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

type RemindersServiceImpl struct {
	repo RemindersRepository
}

func NewRemindersService(repo RemindersRepository) *RemindersServiceImpl {
	return &RemindersServiceImpl{repo: repo}
}

func (s *RemindersServiceImpl) ScheduleForEvent(ctx context.Context, event *events_domain.Event, userID uuid.UUID) error {
	reminders := []*domain.Reminder{
		{
			UserID:     userID,
			EntityType: "event",
			EntityID:   event.ID,
			RemindAt:   event.StartAt.Add(-15 * time.Minute),
		},
		{
			UserID:     userID,
			EntityType: "event",
			EntityID:   event.ID,
			RemindAt:   event.StartAt.Add(-1 * time.Hour),
		},
		{
			UserID:     userID,
			EntityType: "event",
			EntityID:   event.ID,
			RemindAt:   event.StartAt.Add(-24 * time.Hour),
		},
	}

	// Filter out reminders that are in the past
	var validReminders []*domain.Reminder
	now := time.Now().UTC()
	for _, r := range reminders {
		if r.RemindAt.After(now) {
			validReminders = append(validReminders, r)
		}
	}

	if len(validReminders) > 0 {
		return s.repo.CreateReminders(ctx, validReminders)
	}
	return nil
}

func (s *RemindersServiceImpl) ScheduleForTask(ctx context.Context, task *tasks_domain.Task, userID uuid.UUID) error {
	reminders := []*domain.Reminder{
		{
			UserID:     userID,
			EntityType: "task",
			EntityID:   task.ID,
			RemindAt:   task.DueAt.Add(-1 * time.Hour),
		},
		{
			UserID:     userID,
			EntityType: "task",
			EntityID:   task.ID,
			RemindAt:   task.DueAt.Add(-24 * time.Hour),
		},
	}

	// Filter out reminders that are in the past
	var validReminders []*domain.Reminder
	now := time.Now().UTC()
	for _, r := range reminders {
		if r.RemindAt.After(now) {
			validReminders = append(validReminders, r)
		}
	}

	if len(validReminders) > 0 {
		return s.repo.CreateReminders(ctx, validReminders)
	}
	return nil
}

func (s *RemindersServiceImpl) UpdateRemindersForEvent(ctx context.Context, event *events_domain.Event, userID uuid.UUID) error {
	if err := s.DeleteRemindersForEvent(ctx, event.ID); err != nil {
		return err
	}
	return s.ScheduleForEvent(ctx, event, userID)
}

func (s *RemindersServiceImpl) UpdateRemindersForTask(ctx context.Context, task *tasks_domain.Task, userID uuid.UUID) error {
	if err := s.DeleteRemindersForTask(ctx, task.ID); err != nil {
		return err
	}
	return s.ScheduleForTask(ctx, task, userID)
}

func (s *RemindersServiceImpl) DeleteRemindersForEvent(ctx context.Context, eventID uuid.UUID) error {
	return s.repo.DeleteByEntity(ctx, "event", eventID)
}

func (s *RemindersServiceImpl) DeleteRemindersForTask(ctx context.Context, taskID uuid.UUID) error {
	return s.repo.DeleteByEntity(ctx, "task", taskID)
}

func (s *RemindersServiceImpl) ProcessPendingReminders(ctx context.Context, before time.Time, limit int) (int, error) {
	reminders, err := s.repo.GetPending(ctx, before, limit)
	if err != nil {
		return 0, err
	}

	if len(reminders) == 0 {
		return 0, nil
	}

	// Extract IDs to mark as sent
	ids := make([]uuid.UUID, len(reminders))
	for i, r := range reminders {
		ids[i] = r.ID
	}

	// Mark as sent
	if err := s.repo.MarkSent(ctx, ids); err != nil {
		return 0, err
	}

	// TODO: Send actual notifications (in-app, push, email)
	// For now, just mark as sent

	return len(reminders), nil
}

func (s *RemindersServiceImpl) StartWorker(ctx context.Context, interval time.Duration, logger Logger) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				logger.Debug("reminders worker stopped")
				return
			case <-ticker.C:
				processed, err := s.ProcessPendingReminders(ctx, time.Now().UTC().Add(30*time.Second), 100)
				if err != nil {
					logger.Error("failed to process reminders", zap.Error(err))
				} else if processed > 0 {
					logger.Debug("processed reminders", zap.Int("count", processed))
				}
			}
		}
	}()
}