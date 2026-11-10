package domain

import (
	"testing"
	"time"
)

func TestReportCalendarDate(t *testing.T) {
	label, _ := time.Parse("2006-01-02", "2026-11-10")
	date, err := ReportDate(label, "Europe/Moscow")
	if err != nil {
		t.Fatal(err)
	}
	if date.UTC().Format(time.RFC3339) != "2026-11-09T21:00:00Z" {
		t.Fatal(date)
	}
	if _, err := ReportDate(label, "not-a-zone"); err == nil {
		t.Fatal("invalid zone accepted")
	}
}
func TestCalendarDaysAcrossDST(t *testing.T) {
	label, _ := time.Parse("2006-01-02", "2026-03-08")
	day, err := ReportDate(label, "America/New_York")
	if err != nil {
		t.Fatal(err)
	}
	if day.AddDate(0, 0, 1).Sub(day) != 23*time.Hour {
		t.Fatal("calendar boundary must follow DST")
	}
}
