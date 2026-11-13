package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"time"

	"github.com/Aam-Shaegar/Rhythm/internal/features/priorities/domain"
	"github.com/Aam-Shaegar/Rhythm/internal/features/priorities/model"
	tasks "github.com/Aam-Shaegar/Rhythm/internal/features/tasks/domain"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

type Service struct {
	repo  domain.Repository
	model *model.Model
	now   func() time.Time
}

func New(repo domain.Repository, m *model.Model) *Service {
	return &Service{repo: repo, model: m, now: func() time.Time { return time.Now().UTC() }}
}
func input(t *tasks.Task, active, soon int) model.Input {
	desc := ""
	if t.Description != nil {
		desc = *t.Description
	}
	return model.Input{Title: t.Title, Description: desc, DueAt: t.DueAt, CreatedAt: t.CreatedAt, Importance: t.Importance, EstimatedMinutes: t.EstimatedMinutes, Recurring: t.RecurrenceType != nil, ActiveLoad: active, SoonLoad: soon}
}
func revision(t *tasks.Task) string {
	data, _ := json.Marshal(input(t, 0, 0))
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}
func validFeatures(x []float64) bool {
	if len(x) != len(model.FeatureNames) {
		return false
	}
	for _, v := range x {
		if v < 0 || v > 1 || math.IsNaN(v) || math.IsInf(v, 0) {
			return false
		}
	}
	return true
}
func outcome(t *tasks.Task, now time.Time) (bool, bool) {
	if t.IsCompleted {
		if t.CompletedAt == nil {
			return false, false
		}
		return t.CompletedAt.After(t.DueAt), true
	}
	if !t.DueAt.After(now) {
		return true, true
	}
	return false, false
}

type labeled struct {
	TaskID     uuid.UUID
	Revision   string
	Features   []float64
	Late       bool
	ObservedAt time.Time
	OutcomeAt  time.Time
}

func (s *Service) SyncUser(ctx context.Context, uid uuid.UUID) (*domain.Response, error) {
	result := &domain.Response{Items: []domain.Item{}, Recommendations: []uuid.UUID{}, Categories: []domain.CategoryStat{}, ModelVersion: s.model.Version, ModelSource: "synthetic_baseline", PersonalState: "baseline"}
	err := s.repo.WithUser(ctx, uid, func(store domain.Session) error {
		now := s.now()
		list, err := store.Tasks(ctx)
		if err != nil {
			return err
		}
		observations, err := store.Observations(ctx)
		if err != nil {
			return err
		}
		profile, err := store.Profile(ctx)
		if err != nil {
			return err
		}
		active, soon := 0, 0
		for _, t := range list {
			if t.UserID != uid {
				return fmt.Errorf("priority repository crossed account boundary")
			}
			if !t.IsCompleted {
				active++
				if t.DueAt.After(now) && !t.DueAt.After(now.Add(24*time.Hour)) {
					soon++
				}
			}
		}
		data := []labeled{}
		stats := map[string]*domain.CategoryStat{}
		inputs := map[uuid.UUID]model.Input{}
		for _, t := range list {
			otherActive, otherSoon := active, soon
			if !t.IsCompleted {
				otherActive--
				if t.DueAt.After(now) && !t.DueAt.After(now.Add(24*time.Hour)) {
					otherSoon--
				}
			}
			in := input(t, otherActive, otherSoon)
			inputs[t.ID] = in
			rev := revision(t)
			observation, exists := observations[t.ID]
			compatible := exists && observation.Revision == rev && observation.ModelVersion == s.model.Version && validFeatures(observation.Features) && observation.ObservedAt.Before(t.DueAt)
			if !compatible {
				observedAt := s.now()
				if !t.IsCompleted && t.DueAt.After(observedAt) {
					observation = domain.Observation{TaskID: t.ID, Revision: rev, ModelVersion: s.model.Version, Features: s.model.Features(in, observedAt), ObservedAt: observedAt, DueAt: t.DueAt}
					if err := store.SaveObservation(ctx, observation); err != nil {
						return err
					}
					compatible = true
				} else if exists {
					if err := store.DeleteObservation(ctx, t.ID); err != nil {
						return err
					}
				}
			}
			late, known := outcome(t, now)
			if compatible && known && (!t.IsCompleted || t.CompletedAt.After(observation.ObservedAt)) {
				outcomeAt := t.DueAt
				if t.IsCompleted && t.CompletedAt.Before(outcomeAt) {
					outcomeAt = *t.CompletedAt
				}
				data = append(data, labeled{TaskID: t.ID, Revision: rev, Features: observation.Features, Late: late, ObservedAt: observation.ObservedAt, OutcomeAt: outcomeAt})
			}
			category, _ := s.model.Classify(t.Title, in.Description)
			if stats[category] == nil {
				stats[category] = &domain.CategoryStat{Category: category}
			}
			if known {
				stats[category].KnownOutcomes++
				if late {
					stats[category].LateOutcomes++
					if t.IsCompleted {
						stats[category].CompletedLate++
					} else {
						stats[category].OpenLate++
					}
				}
			}
		}
		sort.Slice(data, func(i, j int) bool {
			if data[i].OutcomeAt.Equal(data[j].OutcomeAt) {
				return data[i].TaskID.String() < data[j].TaskID.String()
			}
			return data[i].OutcomeAt.After(data[j].OutcomeAt)
		})
		if len(data) > 500 {
			data = data[:500]
		}
		encoded, _ := json.Marshal(data)
		hash := sha256.Sum256(append([]byte(s.model.Version), encoded...))
		fingerprint := hex.EncodeToString(hash[:])
		if profile.ModelVersion != s.model.Version || profile.DatasetHash != fingerprint || len(profile.Delta) != len(model.FeatureNames) {
			examples := make([]model.Example, len(data))
			for i, e := range data {
				examples[i] = model.Example{Features: e.Features, Late: e.Late}
			}
			profile = domain.Profile{ModelVersion: s.model.Version, DatasetHash: fingerprint, Delta: s.model.FitPersonal(examples), Samples: len(examples), TrainedAt: now}
			if err := store.SaveProfile(ctx, profile); err != nil {
				return err
			}
		}
		result.PersonalSamples = profile.Samples
		result.TrainedAt = profile.TrainedAt
		if profile.Samples > 0 {
			result.ModelSource = "synthetic_baseline_with_personal_history"
			result.PersonalState = "learning"
		}
		if profile.Samples >= 5 {
			result.PersonalState = "personalized"
		}
		for _, t := range list {
			if !t.IsCompleted {
				result.Items = append(result.Items, domain.Item{TaskID: t.ID, Prediction: s.model.Predict(inputs[t.ID], now, profile.Delta, profile.Samples)})
			}
		}
		due := map[uuid.UUID]time.Time{}
		for _, t := range list {
			due[t.ID] = t.DueAt
		}
		sort.Slice(result.Items, func(i, j int) bool {
			a, b := result.Items[i], result.Items[j]
			if a.Score != b.Score {
				return a.Score > b.Score
			}
			if !due[a.TaskID].Equal(due[b.TaskID]) {
				return due[a.TaskID].Before(due[b.TaskID])
			}
			return a.TaskID.String() < b.TaskID.String()
		})
		if len(result.Items) > 0 {
			highest := result.Items[0].Level
			for _, item := range result.Items {
				if len(result.Recommendations) >= 3 || item.Level != highest {
					break
				}
				result.Recommendations = append(result.Recommendations, item.TaskID)
			}
		}
		for _, stat := range stats {
			if stat.KnownOutcomes > 0 {
				stat.LateRate = float64(stat.LateOutcomes) / float64(stat.KnownOutcomes)
			}
			result.Categories = append(result.Categories, *stat)
		}
		sort.Slice(result.Categories, func(i, j int) bool {
			a, b := result.Categories[i], result.Categories[j]
			if a.LateRate != b.LateRate {
				return a.LateRate > b.LateRate
			}
			if a.KnownOutcomes != b.KnownOutcomes {
				return a.KnownOutcomes > b.KnownOutcomes
			}
			return a.Category < b.Category
		})
		return nil
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

type Logger interface{ Warn(string, ...zap.Field) }

func (s *Service) StartWorker(ctx context.Context, interval time.Duration, log Logger) {
	go func() {
		run := func() {
			ids, err := s.repo.UserIDs(ctx)
			if err != nil {
				if ctx.Err() == nil {
					log.Warn("priority worker user lookup failed", zap.Error(err))
				}
				return
			}
			for _, id := range ids {
				if ctx.Err() != nil {
					return
				}
				userCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
				_, err := s.SyncUser(userCtx, id)
				cancel()
				if err != nil {
					log.Warn("priority worker sync failed", zap.String("user_id", id.String()), zap.Error(err))
				}
			}
		}
		run()
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				run()
			}
		}
	}()
}
