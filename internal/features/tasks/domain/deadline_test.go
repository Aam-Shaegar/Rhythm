package domain

import (
	"testing"
	"time"
)

func TestDeadlineStatus(t *testing.T) {
	due := time.Date(2026, 11, 10, 12, 0, 0, 0, time.UTC)
	for _, tc := range []struct {
		name          string
		done          bool
		delta         time.Duration
		late, overdue bool
	}{
		{"unfinished before deadline", false, -time.Second, false, false},
		{"unfinished after deadline", false, time.Second, false, true},
		{"completed at deadline", true, 0, false, false},
		{"completed early", true, -time.Second, false, false},
		{"completed late", true, time.Second, true, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			at := due.Add(tc.delta)
			task := Task{DueAt: due, IsCompleted: tc.done}
			if tc.done {
				task.CompletedAt = &at
			}
			if task.CompletedLate() != tc.late || task.OverdueAt(at) != tc.overdue {
				t.Fatalf("wrong status: %+v", task)
			}
		})
	}
}
