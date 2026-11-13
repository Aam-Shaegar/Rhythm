package service

import (
	"context"
	"github.com/Aam-Shaegar/Rhythm/internal/features/priorities/domain"
	"github.com/Aam-Shaegar/Rhythm/internal/features/priorities/model"
	tasks "github.com/Aam-Shaegar/Rhythm/internal/features/tasks/domain"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"reflect"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

type memoryUser struct {
	tasks        []*tasks.Task
	observations map[uuid.UUID]domain.Observation
	profile      domain.Profile
	writes       int
}

func (u *memoryUser) Tasks(context.Context) ([]*tasks.Task, error) { return u.tasks, nil }
func (u *memoryUser) Observations(context.Context) (map[uuid.UUID]domain.Observation, error) {
	return u.observations, nil
}
func (u *memoryUser) Profile(context.Context) (domain.Profile, error) { return u.profile, nil }
func (u *memoryUser) SaveObservation(_ context.Context, o domain.Observation) error {
	u.observations[o.TaskID] = o
	return nil
}
func (u *memoryUser) DeleteObservation(_ context.Context, id uuid.UUID) error {
	delete(u.observations, id)
	return nil
}
func (u *memoryUser) SaveProfile(_ context.Context, p domain.Profile) error {
	u.profile = p
	u.writes++
	return nil
}

type memoryRepo struct {
	mu    sync.Mutex
	users map[uuid.UUID]*memoryUser
}

func (r *memoryRepo) WithUser(_ context.Context, id uuid.UUID, fn func(domain.Session) error) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	return fn(r.users[id])
}
func (r *memoryRepo) UserIDs(context.Context) ([]uuid.UUID, error) {
	ids := []uuid.UUID{}
	for id := range r.users {
		ids = append(ids, id)
	}
	return ids, nil
}
func setup(t *testing.T) (*Service, *memoryRepo, *memoryUser, uuid.UUID, *time.Time) {
	t.Helper()
	uid := uuid.New()
	now := time.Date(2026, 11, 13, 12, 0, 0, 0, time.UTC)
	u := &memoryUser{observations: map[uuid.UUID]domain.Observation{}}
	r := &memoryRepo{users: map[uuid.UUID]*memoryUser{uid: u}}
	m, err := model.Load()
	if err != nil {
		t.Fatal(err)
	}
	s := New(r, m)
	s.now = func() time.Time { return now }
	return s, r, u, uid, &now
}
func task(uid uuid.UUID, now time.Time) *tasks.Task {
	return &tasks.Task{ID: uuid.New(), UserID: uid, Title: "Подготовить отчет проекта", CreatedAt: now.Add(-time.Hour), DueAt: now.Add(time.Hour), Importance: 2, EstimatedMinutes: 30}
}
func mustSync(t *testing.T, s *Service, uid uuid.UUID) *domain.Response {
	t.Helper()
	result, err := s.SyncUser(context.Background(), uid)
	if err != nil {
		t.Fatal(err)
	}
	return result
}
func TestRepeatedViewsDoNotTrainOrReplaceSnapshot(t *testing.T) {
	s, _, u, uid, now := setup(t)
	a := task(uid, *now)
	u.tasks = []*tasks.Task{a}
	r := mustSync(t, s, uid)
	if r.PersonalSamples != 0 || len(u.observations) != 1 || len(r.Recommendations) != 1 {
		t.Fatal(r)
	}
	saved := u.observations[a.ID]
	trained := u.profile.TrainedAt
	*now = now.Add(10 * time.Minute)
	for i := 0; i < 10; i++ {
		mustSync(t, s, uid)
	}
	if u.writes != 1 || !reflect.DeepEqual(saved, u.observations[a.ID]) || !trained.Equal(u.profile.TrainedAt) {
		t.Fatal("viewing counted as training")
	}
}
func TestLateCompletionTransferReopenAndDeleteRetractLabels(t *testing.T) {
	s, _, u, uid, now := setup(t)
	a := task(uid, *now)
	u.tasks = []*tasks.Task{a}
	mustSync(t, s, uid)
	*now = now.Add(2 * time.Hour)
	r := mustSync(t, s, uid)
	if r.PersonalSamples != 1 || r.Categories[0].OpenLate != 1 || r.Items[0].Risk != 1 {
		t.Fatal("missing mature outcome", r)
	}
	mustSync(t, s, uid)
	if u.writes != 2 {
		t.Fatal("duplicate learning")
	}
	finished := *now
	a.IsCompleted = true
	a.CompletedAt = &finished
	r = mustSync(t, s, uid)
	if r.PersonalSamples != 1 || len(r.Items) != 0 || r.Categories[0].CompletedLate != 1 {
		t.Fatal(r)
	}
	a.IsCompleted = false
	a.CompletedAt = nil
	a.DueAt = now.Add(time.Hour)
	r = mustSync(t, s, uid)
	if r.PersonalSamples != 0 || len(r.Items) != 1 {
		t.Fatal("deadline move did not retract prior label", r)
	}
	*now = now.Add(30 * time.Minute)
	finished = *now
	a.IsCompleted = true
	a.CompletedAt = &finished
	r = mustSync(t, s, uid)
	if r.PersonalSamples != 1 || r.Categories[0].LateOutcomes != 0 {
		t.Fatal("on-time label wrong", r)
	}
	a.IsCompleted = false
	a.CompletedAt = nil
	r = mustSync(t, s, uid)
	if r.PersonalSamples != 0 {
		t.Fatal("reopening should retract future outcome")
	}
	*now = now.Add(time.Hour)
	mustSync(t, s, uid)
	u.tasks = nil
	delete(u.observations, a.ID)
	r = mustSync(t, s, uid)
	if r.PersonalSamples != 0 || len(r.Recommendations) != 0 {
		t.Fatal("deleted outcome retained", r)
	}
}
func TestNoHistoricalLeakageAndCompletedTasksNeverRecommended(t *testing.T) {
	s, _, u, uid, now := setup(t)
	old := task(uid, *now)
	old.DueAt = now.Add(-time.Hour)
	done := *now
	old.IsCompleted = true
	old.CompletedAt = &done
	future := task(uid, *now)
	future.Title = "zxqv"
	u.tasks = []*tasks.Task{old, future}
	r := mustSync(t, s, uid)
	if r.PersonalSamples != 0 || len(u.observations) != 1 || len(r.Items) != 1 || r.Items[0].TaskID != future.ID {
		t.Fatal(r)
	}
	if u.observations[future.ID].ObservedAt.After(*now) {
		t.Fatal("bad observation clock")
	}
}
func TestUsersAreIsolatedAndConcurrentViewsIdempotent(t *testing.T) {
	s, r, a, uid, now := setup(t)
	other := uuid.New()
	b := &memoryUser{observations: map[uuid.UUID]domain.Observation{}}
	r.users[other] = b
	a.tasks = []*tasks.Task{task(uid, *now)}
	b.tasks = []*tasks.Task{task(other, *now)}
	mustSync(t, s, uid)
	mustSync(t, s, other)
	*now = now.Add(2 * time.Hour)
	a.tasks[0].IsCompleted = true
	done := *now
	a.tasks[0].CompletedAt = &done
	b.tasks[0].DueAt = now.Add(time.Hour)
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := s.SyncUser(context.Background(), uid); err != nil {
				t.Error(err)
			}
		}()
	}
	wg.Wait()
	mine := mustSync(t, s, uid)
	theirs := mustSync(t, s, other)
	if mine.PersonalSamples != 1 || theirs.PersonalSamples != 0 || a.writes != 2 {
		t.Fatal("account leakage or duplicate training")
	}
	for _, v := range b.profile.Delta {
		if v != 0 {
			t.Fatal("weights crossed accounts")
		}
	}
}
func TestModelUpgradeDiscardsIncompatibleSnapshots(t *testing.T) {
	s, _, u, uid, now := setup(t)
	a := task(uid, *now)
	u.tasks = []*tasks.Task{a}
	mustSync(t, s, uid)
	*now = now.Add(2 * time.Hour)
	mustSync(t, s, uid)
	s.model.Version = "priority-next"
	r := mustSync(t, s, uid)
	if r.PersonalSamples != 0 || u.profile.ModelVersion != "priority-next" || len(u.observations) != 0 {
		t.Fatal("incompatible model data reused")
	}
}

func TestTrainingWindowIsBounded(t *testing.T) {
	s, _, u, uid, now := setup(t)
	for i := 0; i < 510; i++ {
		u.tasks = append(u.tasks, task(uid, *now))
	}
	mustSync(t, s, uid)
	*now = now.Add(2 * time.Hour)
	r := mustSync(t, s, uid)
	if r.PersonalSamples != 500 || r.Categories[0].KnownOutcomes != 510 {
		t.Fatal("training window or history count is wrong", r.PersonalSamples)
	}
}

type silentLogger struct{}

func (silentLogger) Warn(string, ...zap.Field) {}
func TestWorkerLearnsDeadlineWithoutAView(t *testing.T) {
	s, r, u, uid, now := setup(t)
	a := task(uid, *now)
	u.tasks = []*tasks.Task{a}
	mustSync(t, s, uid)
	var clock atomic.Int64
	clock.Store(now.UnixNano())
	s.now = func() time.Time { return time.Unix(0, clock.Load()).UTC() }
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	s.StartWorker(ctx, 5*time.Millisecond, silentLogger{})
	clock.Store(now.Add(2 * time.Hour).UnixNano())
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		r.mu.Lock()
		samples := u.profile.Samples
		r.mu.Unlock()
		if samples == 1 {
			return
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatal("background worker did not learn deadline without a request")
}
