package domain

import (
	"time"
)

type DailyReport struct {
	CompletedOnTime int       `json:"completed_on_time"`
	CompletedLate   int       `json:"completed_late"`
	OverdueTasks    int       `json:"overdue_tasks"`
	OnTimePct       float64   `json:"on_time_pct"`
	Date            time.Time `json:"date"`
	TotalTasks      int       `json:"total_tasks"`
	CompletedTasks  int       `json:"completed_tasks"`
	CompletionPct   float64   `json:"completion_pct"`
	EventsCount     int       `json:"events_count"`
}

type DailyReportQuery struct {
	TimeZone string    `query:"time_zone"`
	Date     time.Time `query:"date"`
}

type PeriodReportQuery struct {
	TimeZone string    `query:"time_zone"`
	From     time.Time `query:"from"`
	To       time.Time `query:"to"`
}
