package service

import (
	"context"
	"time"

	"go.uber.org/zap"
)

func (s *JwtServiceImpl) StartCleanup(ctx context.Context, interval time.Duration, logger Logger) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				logger.Debug("jwt cleanup worker stopped")
				return
			case <-ticker.C:
				deleted, err := s.jwtRepo.DeleteExpiredTokens(ctx)
				if err != nil {
					logger.Error("failed to cleanup expired refresh tokens", zap.Error(err))
				} else if deleted > 0 {
					logger.Debug("cleaned up expired refresh tokens", zap.Int64("count", deleted))
				}
			}
		}
	}()
}
