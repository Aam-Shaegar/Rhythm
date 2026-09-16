package domain

import (
	"time"

	"github.com/google/uuid"
)

type EventFilter struct {
	UserID   uuid.UUID  `query:"-"`
	DateFrom *time.Time `query:"date_from"`
	DateTo   *time.Time `query:"date_to"`
	View     string     `query:"view" validate:"omitempty,oneof=day week month"`
	Limit    int        `query:"limit" validate:"omitempty,min=0,max=1000"`
	Offset   int        `query:"offset" validate:"omitempty,min=0"`
}