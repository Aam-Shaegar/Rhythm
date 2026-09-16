package domain

import (
	"time"

	"github.com/google/uuid"
)

type DailyReport struct {
	Date            time.Time `json:"date"`
	TotalTasks      int       `json:"total_tasks"`
	CompletedTasks  int       `json:"completed_tasks"`
	CompletionPct   float64   `json:"completion_pct"`
	EventsCount     int       `json:"events_count"`
}

type DailyReportQuery struct {
	Date time.Time `query:"date"`
}

type PeriodReportQuery struct {
	From time.Time `query:"from"`
	To   time.Time `query:"to"`
}

type ReportFilter struct {
	UserID   uuid.UUID
	DateFrom time.Time
	DateTo   time.Time
}

type ReportResponse struct {
	Date            string  `json:"date"`
	TotalTasks      int     `json:"total_tasks"`
	CompletedTasks  int     `json:"completed_tasks"`
	CompletionPct   float64 `json:"completion_pct"`
	EventsCount     int     `json:"events_count"`
}