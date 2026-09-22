package service

import (
	"context"
	"errors"
	"net/url"
	"time"

	"go.uber.org/zap"

	events_domain "github.com/Aam-Shaegar/Rhythm/internal/features/events/domain"
	tasks_domain "github.com/Aam-Shaegar/Rhythm/internal/features/tasks/domain"
	"github.com/Aam-Shaegar/Rhythm/internal/features/reminders/domain"
	"github.com/google/uuid"
)

type RemindersServiceImpl struct {
	repo   RemindersRepository
	sender PushSender
	log    Logger
}

// sender may be nil (no VAPID keys configured) — scheduling and the
// settings API keep working, only push delivery is skipped.
func NewRemindersService(repo RemindersRepository, sender PushSender) *RemindersServiceImpl {
	return &RemindersServiceImpl{repo: repo, sender: sender}
}

// SetLogger enables delivery diagnostics. Nil disables logging
// (used in tests); main.go passes the app logger.
func (s *RemindersServiceImpl) SetLogger(l Logger) {
	s.log = l
}

func (s *RemindersServiceImpl) ScheduleForEvent(ctx context.Context, event *events_domain.Event, userID uuid.UUID) error {
	reminders := []*domain.Reminder{
		{
			UserID:     userID,
			EntityType: "event",
			EntityID:   event.ID,
			Title:      event.Title,
			RemindAt:   event.StartAt.Add(-15 * time.Minute),
		},
		{
			UserID:     userID,
			EntityType: "event",
			EntityID:   event.ID,
			Title:      event.Title,
			RemindAt:   event.StartAt.Add(-1 * time.Hour),
		},
		{
			UserID:     userID,
			EntityType: "event",
			EntityID:   event.ID,
			Title:      event.Title,
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
			Title:      task.Title,
			RemindAt:   task.DueAt.Add(-1 * time.Hour),
		},
		{
			UserID:     userID,
			EntityType: "task",
			EntityID:   task.ID,
			Title:      task.Title,
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

	// Best-effort push delivery: failures never fail the batch,
	// expired endpoints are cleaned up silently.
	s.deliverPushes(ctx, reminders)

	return len(reminders), nil
}

// deliverPushes groups reminders by user, loads each user's push
// subscriptions once, and sends every reminder to every device.
func (s *RemindersServiceImpl) deliverPushes(ctx context.Context, reminders []*domain.Reminder) {
	if s.sender == nil {
		return
	}

	byUser := make(map[uuid.UUID][]*domain.Reminder)
	for _, r := range reminders {
		byUser[r.UserID] = append(byUser[r.UserID], r)
	}

	for userID, rs := range byUser {
		subs, err := s.repo.GetSubscriptionsByUser(ctx, userID)
		if err != nil {
			s.warn("push subscriptions lookup failed", zap.String("user_id", userID.String()), zap.Error(err))
			continue
		}
		if len(subs) == 0 {
			s.debug("no push subscriptions, skipping delivery", zap.String("user_id", userID.String()), zap.Int("reminders", len(rs)))
			continue
		}
		for _, r := range rs {
			payload := BuildPushPayload(r)
			for _, sub := range subs {
				if err := s.sender.Send(ctx, sub, payload); err != nil {
					if errors.Is(err, ErrSubscriptionGone) {
						s.debug("push endpoint expired, deleting subscription",
							zap.String("user_id", userID.String()),
							zap.String("endpoint_host", endpointHost(sub.Endpoint)))
						_ = s.repo.DeleteSubscriptionByEndpoint(ctx, userID, sub.Endpoint)
					} else {
						s.warn("push delivery failed",
							zap.String("user_id", userID.String()),
							zap.String("endpoint_host", endpointHost(sub.Endpoint)),
							zap.String("reminder_id", r.ID.String()),
							zap.Error(err))
					}
				} else {
					s.debug("push delivered",
						zap.String("user_id", userID.String()),
						zap.String("endpoint_host", endpointHost(sub.Endpoint)),
						zap.String("reminder_id", r.ID.String()))
				}
			}
		}
	}
}

func (s *RemindersServiceImpl) SavePushSubscription(ctx context.Context, userID uuid.UUID, input domain.PushSubscriptionInput) error {	return s.repo.UpsertSubscription(ctx, &domain.PushSubscription{
		UserID:   userID,
		Endpoint: input.Endpoint,
		P256DH:   input.P256DH,
		Auth:     input.Auth,
	})
}

func (s *RemindersServiceImpl) DeletePushSubscription(ctx context.Context, userID uuid.UUID, endpoint string) error {
	return s.repo.DeleteSubscriptionByEndpoint(ctx, userID, endpoint)
}

// endpointHost extracts "host" from a push endpoint for logs.
// Full endpoints contain secrets, never log them whole.
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