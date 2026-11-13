package domain

import (
	"context"
	"github.com/Aam-Shaegar/Rhythm/internal/features/priorities/model"
	tasks "github.com/Aam-Shaegar/Rhythm/internal/features/tasks/domain"
	"github.com/google/uuid"
	"time"
)

type Observation struct {
	TaskID                 uuid.UUID
	Revision, ModelVersion string
	Features               []float64
	ObservedAt, DueAt      time.Time
}
type Profile struct {
	ModelVersion, DatasetHash string
	Delta                     []float64
	Samples                   int
	TrainedAt                 time.Time
}
type Item struct {
	TaskID uuid.UUID `json:"task_id"`
	model.Prediction
}
type CategoryStat struct {
	Category      string  `json:"category"`
	KnownOutcomes int     `json:"known_outcomes"`
	LateOutcomes  int     `json:"late_outcomes"`
	CompletedLate int     `json:"completed_late"`
	OpenLate      int     `json:"open_late"`
	LateRate      float64 `json:"late_rate"`
}
type Response struct {
	Items           []Item         `json:"items"`
	Recommendations []uuid.UUID    `json:"recommendations"`
	Categories      []CategoryStat `json:"categories"`
	ModelVersion    string         `json:"model_version"`
	ModelSource     string         `json:"model_source"`
	PersonalSamples int            `json:"personal_samples"`
	PersonalState   string         `json:"personal_state"`
	TrainedAt       time.Time      `json:"trained_at"`
}
type Session interface {
	Tasks(context.Context) ([]*tasks.Task, error)
	Observations(context.Context) (map[uuid.UUID]Observation, error)
	Profile(context.Context) (Profile, error)
	SaveObservation(context.Context, Observation) error
	DeleteObservation(context.Context, uuid.UUID) error
	SaveProfile(context.Context, Profile) error
}
type Repository interface {
	WithUser(context.Context, uuid.UUID, func(Session) error) error
	UserIDs(context.Context) ([]uuid.UUID, error)
}
