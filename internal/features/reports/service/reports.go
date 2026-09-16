package service

import (
	"context"
	"time"

	"github.com/Aam-Shaegar/Rhythm/internal/features/reports/domain"
	tasks_domain "github.com/Aam-Shaegar/Rhythm/internal/features/tasks/domain"
	events_domain "github.com/Aam-Shaegar/Rhythm/internal/features/events/domain"
	tasks_postgres "github.com/Aam-Shaegar/Rhythm/internal/features/tasks/repository/postgres"
	events_postgres "github.com/Aam-Shaegar/Rhythm/internal/features/events/repository/postgres"
	"github.com/google/uuid"
)

type ReportsServiceImpl struct {
	tasksRepo  tasks_postgres.TasksRepository
	eventsRepo events_postgres.EventsRepository
}

func NewReportsService(tasksRepo tasks_postgres.TasksRepository, eventsRepo events_postgres.EventsRepository) *ReportsServiceImpl {
	return &ReportsServiceImpl{
		tasksRepo:  tasksRepo,
		eventsRepo: eventsRepo,
	}
}

func (s *ReportsServiceImpl) GetDailyReport(ctx context.Context, userID uuid.UUID, date time.Time) (*domain.DailyReport, error) {
	startOfDay := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, date.Location())
	endOfDay := startOfDay.Add(24 * time.Hour)

	tasks, err := s.tasksRepo.GetByUserID(ctx, tasks_domain.TaskFilter{
		UserID:    userID,
		DateFrom:  &startOfDay,
		DateTo:    &endOfDay,
	})
	if err != nil {
		return nil, err
	}

	events, err := s.eventsRepo.GetByUserID(ctx, events_domain.EventFilter{
		UserID:   userID,
		DateFrom: &startOfDay,
		DateTo:   &endOfDay,
	})
	if err != nil {
		return nil, err
	}

	totalTasks := len(tasks)
	completedTasks := 0
	for _, t := range tasks {
		if t.IsCompleted {
			completedTasks++
		}
	}

	var completionPct float64
	if totalTasks > 0 {
		completionPct = float64(completedTasks) / float64(totalTasks) * 100
	}

	return &domain.DailyReport{
		Date:            date,
		TotalTasks:      totalTasks,
		CompletedTasks:  completedTasks,
		CompletionPct:   completionPct,
		EventsCount:     len(events),
	}, nil
}

func (s *ReportsServiceImpl) GetPeriodReport(ctx context.Context, userID uuid.UUID, from, to time.Time) ([]*domain.DailyReport, error) {
	var reports []*domain.DailyReport

	current := from
	for !current.After(to) {
		report, err := s.GetDailyReport(ctx, userID, current)
		if err != nil {
			return nil, err
		}
		reports = append(reports, report)
		current = current.Add(24 * time.Hour)
	}

	return reports, nil
}