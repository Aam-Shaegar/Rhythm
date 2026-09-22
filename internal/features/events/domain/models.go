package domain

import (
	"time"

	"github.com/google/uuid"
)

type Event struct {
	ID          uuid.UUID
	UserID      uuid.UUID
	Title       string
	Description *string
	StartAt     time.Time
	EndAt       time.Time
	CreatedAt   time.Time
	UpdatedAt   time.Time
}
