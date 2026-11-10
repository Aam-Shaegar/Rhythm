package service

import (
	"context"
	"errors"
	"fmt"
	core_errors "github.com/Aam-Shaegar/Rhythm/internal/core/errors"
	"net/url"
	"time"

	"go.uber.org/zap"

	events_domain "github.com/Aam-Shaegar/Rhythm/internal/features/events/domain"
	"github.com/Aam-Shaegar/Rhythm/internal/features/reminders/domain"
	tasks_domain "github.com/Aam-Shaegar/Rhythm/internal/features/tasks/domain"
	"github.com/google/uuid"
)

type RemindersServiceImpl struct {
	repo   RemindersRepository
	sender PushSender
	log    Logger
}

// sender nil = только планирование, без push-доставки.
func NewRemindersService(repo RemindersRepository, sender PushSender) *RemindersServiceImpl {
	return &RemindersServiceImpl{repo: repo, sender: sender}
}

// Nil отключает логирование.
func (s *RemindersServiceImpl) SetLogger(l Logger) {
	s.log = l
}

func (s *RemindersServiceImpl) GetSettings(ctx context.Context, uid uuid.UUID) (domain.ReminderSettings, error) {
	if repo, ok := s.repo.(SettingsRepository); ok {
		return repo.GetSettings(ctx, uid)
	}
	return domain.DefaultSettings(), nil
}

func (s *RemindersServiceImpl) UpdateSettings(ctx context.Context, uid uuid.UUID, input domain.UpdateReminderSettingsInput) (domain.ReminderSettings, error) {
	if err := input.Validate(); err != nil {
		return domain.ReminderSettings{}, fmt.Errorf("%v: %w", err, core_errors.ErrInvalidArgument)
	}
	repo, ok := s.repo.(SettingsRepository)
	if !ok {
		return domain.ReminderSettings{}, errors.New("settings repository unavailable")
	}
	return repo.SaveSettings(ctx, uid, input)
}

func (s *RemindersServiceImpl) schedule(ctx context.Context, uid, id uuid.UUID, kind, title string, at time.Time, offsets []int) error {
	var reminders []*domain.Reminder
	now := time.Now().UTC()
	for _, offset := range offsets {
		remindAt := at.Add(-time.Duration(offset) * time.Minute)
		if remindAt.After(now) {
			reminders = append(reminders, &domain.Reminder{UserID: uid, EntityID: id, EntityType: kind, Title: title, RemindAt: remindAt})
		}
	}
	return s.repo.CreateReminders(ctx, reminders)
}
func (s *RemindersServiceImpl) ScheduleForEvent(ctx context.Context, event *events_domain.Event, uid uuid.UUID) error {
	settings, err := s.GetSettings(ctx, uid)
	if err != nil {
		return err
	}
	return s.schedule(ctx, uid, event.ID, "event", event.Title, event.StartAt, settings.EventOffsets)
}
func (s *RemindersServiceImpl) ScheduleForTask(ctx context.Context, task *tasks_domain.Task, uid uuid.UUID) error {
	if task.IsCompleted {
		return nil
	}
	settings, err := s.GetSettings(ctx, uid)
	if err != nil {
		return err
	}
	return s.schedule(ctx, uid, task.ID, "task", task.Title, task.DueAt, settings.TaskOffsets)
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
	if repo, ok := s.repo.(SettingsRepository); ok {
		if err := repo.QueueOverdue(ctx, before); err != nil {
			return 0, err
		}
	}
	reminders, err := s.repo.GetPending(ctx, before, limit)
	if err != nil {
		return 0, err
	}

	if len(reminders) == 0 {
		return 0, nil
	}

	// Only acknowledge deliveries after a successful push. Claimed rows retry after lease expiry.
	for _, rem := range reminders {
		if s.deliverPush(ctx, rem) {
			if err := s.repo.MarkSent(ctx, []uuid.UUID{rem.ID}); err != nil {
				return 0, err
			}
		}
	}
	return len(reminders), nil
}

func (s *RemindersServiceImpl) deliverPush(ctx context.Context, rem *domain.Reminder) bool {
	if s.sender == nil {
		return false
	}
	subs, err := s.repo.GetSubscriptionsByUser(ctx, rem.UserID)
	if err != nil {
		s.warn("push subscriptions lookup failed", zap.Error(err))
		return false
	}
	// No devices subscribed: nothing to deliver. Future subscriptions receive future reminders.
	success := true
	for _, sub := range subs {
		if err := s.sender.Send(ctx, sub, BuildPushPayload(rem)); err != nil {
			if errors.Is(err, ErrSubscriptionGone) {
				_ = s.repo.DeleteSubscriptionByEndpoint(ctx, rem.UserID, sub.Endpoint)
			} else {
				s.warn("push delivery failed", zap.Error(err), zap.String("reminder_id", rem.ID.String()))
				success = false
			}
		}
	}
	return success
}

func (s *RemindersServiceImpl) SavePushSubscription(ctx context.Context, userID uuid.UUID, input domain.PushSubscriptionInput) error {
	return s.repo.UpsertSubscription(ctx, &domain.PushSubscription{
		UserID:   userID,
		Endpoint: input.Endpoint,
		P256DH:   input.P256DH,
		Auth:     input.Auth,
	})
}

func (s *RemindersServiceImpl) DeletePushSubscription(ctx context.Context, userID uuid.UUID, endpoint string) error {
	return s.repo.DeleteSubscriptionByEndpoint(ctx, userID, endpoint)
}

// В логи только host: полный endpoint содержит секреты.
func endpointHost(endpoint string) string {
	u, err := url.Parse(endpoint)
	if err != nil || u.Host == "" {
		return "unknown"
	}
	return u.Host
}

func (s *RemindersServiceImpl) warn(msg string, fields ...zap.Field) {
	if s.log != nil {
		s.log.Warn(msg, fields...)
	}
}

func (s *RemindersServiceImpl) debug(msg string, fields ...zap.Field) {
	if s.log != nil {
		s.log.Debug(msg, fields...)
	}
}

func (s *RemindersServiceImpl) PushPublicKey() string {
	if w, ok := s.sender.(*WebPushSender); ok {
		return w.PublicKey
	}
	return ""
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
				processed, err := s.ProcessPendingReminders(ctx, time.Now().UTC(), 100)
				if err != nil {
					logger.Error("failed to process reminders", zap.Error(err))
				} else if processed > 0 {
					logger.Debug("processed reminders", zap.Int("count", processed))
				}
			}
		}
	}()
}
