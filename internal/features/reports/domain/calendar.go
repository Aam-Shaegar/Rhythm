package domain

import (
	"fmt"
	core_errors "github.com/Aam-Shaegar/Rhythm/internal/core/errors"
	"time"
)

// ReportDate treats date-only input as a calendar label in the viewer's time zone.
func ReportDate(date time.Time, zone string) (time.Time, error) {
	loc := time.Local
	if zone != "" {
		var err error
		loc, err = time.LoadLocation(zone)
		if err != nil {
			return time.Time{}, fmt.Errorf("invalid time zone: %w", core_errors.ErrInvalidArgument)
		}
	}
	if date.IsZero() {
		date = time.Now().In(loc)
	}
	return time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, loc), nil
}
