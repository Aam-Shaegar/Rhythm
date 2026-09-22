package domain

import (
	"time"

	"github.com/google/uuid"
)

type Reminder struct {
	UserID     uuid.UUID
	EntityType string
	EntityID   uuid.UUID
	RemindAt   time.Time
}
