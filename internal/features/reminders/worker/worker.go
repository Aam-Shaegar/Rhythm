package worker

import (
	"context"
	"time"

	"github.com/Aam-Shaegar/Rhythm/internal/features/reminders/service"
	"go.uber.org/zap"
)

type ReminderWorker struct {
	svc    service.RemindersService
	logger *zap.Logger
}

func NewReminderWorker(svc service.RemindersService, logger *zap.Logger) *ReminderWorker {
	return &ReminderWorker{
		svc:    svc,
		logger: logger,
	}
}

func (w *ReminderWorker) Run(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			w.logger.Debug("reminder worker stopped")
			return
		case <-ticker.C:
			processed, err := w.svc.ProcessPendingReminders(ctx, time.Now().UTC().Add(30*time.Second), 100)
			if err != nil {
				w.logger.Error("failed to process reminders", zap.Error(err))
			} else if processed > 0 {
				w.logger.Debug("processed reminders", zap.Int("count", processed))
			}
		}
	}
}