package service

import (
	"context"
	"time"

	"github.com/Aam-Shaegar/Rhythm/internal/features/reports/domain"
	"github.com/google/uuid"
)

type ReportsService interface {
	GetDailyReport(ctx context.Context, userID uuid.UUID, date time.Time) (*domain.DailyReport, error)
	GetPeriodReport(ctx context.Context, userID uuid.UUID, from, to time.Time) ([]*domain.DailyReport, error)
}