package service

import (
	"context"
	"errors"
	"testing"
	"time"

	events_domain "github.com/Aam-Shaegar/Rhythm/internal/features/events/domain"
	"github.com/Aam-Shaegar/Rhythm/internal/features/reminders/domain"
	tasks_domain "github.com/Aam-Shaegar/Rhythm/internal/features/tasks/domain"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

type mockRemindersRepo struct {
	created [][]*domain.Reminder
	pending []*domain.Reminder
	marked  [][]uuid.UUID
	deleted [][2]string // entityType + id string (simplified)
	err     error
}

func (m *mockRemindersRepo) CreateReminders(ctx context.Context, rs []*domain.Reminder) error {
	if m.err != nil {
		return m.err
	}
	m.created = append(m.created, rs)
	return nil
}
func (m *mockRemindersRepo) GetPending(ctx context.Context, before time.Time, limit int) ([]*domain.Reminder, error) {
	if m.err != nil {
		return nil, m.err
	}
	if len(m.pending) > limit {
		return m.pending[:limit], nil
	}
	return m.pending, nil
}
func (m *mockRemindersRepo) MarkSent(ctx context.Context, ids []uuid.UUID) error {
	if m.err != nil {
		return m.err
	}
	m.marked = append(m.marked, ids)
	return nil
}
func (m *mockRemindersRepo) GetByEntity(ctx context.Context, t string, id uuid.UUID) ([]*domain.Reminder, error) {
	return nil, nil
}
func (m *mockRemindersRepo) DeleteByEntity(ctx context.Context, t string, id uuid.UUID) error {
	if m.err != nil {
		return m.err
	}
	m.deleted = append(m.deleted, [2]string{t, id.String()})
	return nil
}
func (m *mockRemindersRepo) UpsertSubscription(ctx context.Context, sub *domain.PushSubscription) error {
	if m.err != nil {
		return m.err
	}
	return nil
}
func (m *mockRemindersRepo) GetSubscriptionsByUser(ctx context.Context, userID uuid.UUID) ([]*domain.PushSubscription, error) {
	if m.err != nil {
		return nil, m.err
	}
	return nil, nil
}
func (m *mockRemindersRepo) DeleteSubscriptionByEndpoint(ctx context.Context, userID uuid.UUID, endpoint string) error {
	if m.err != nil {
		return m.err
	}
	return nil
}

func TestScheduleForEvent_FutureCreatesThree(t *testing.T) {
	repo := &mockRemindersRepo{}
	svc := NewRemindersService(repo, nil)
	ev := &events_domain.Event{ID: uuid.New(), StartAt: time.Now().UTC().Add(48 * time.Hour)}
	if err := svc.ScheduleForEvent(context.Background(), ev, uuid.New()); err != nil {
		t.Fatal(err)
	}
	if len(repo.created) != 1 || len(repo.created[0]) != 3 {
		t.Fatalf("expected 3 reminders, got %v", repo.created)
	}
}

func TestScheduleForEvent_PastCreatesNone(t *testing.T) {
	repo := &mockRemindersRepo{}
	svc := NewRemindersService(repo, nil)
	ev := &events_domain.Event{ID: uuid.New(), StartAt: time.Now().UTC().Add(-48 * time.Hour)}
	if err := svc.ScheduleForEvent(context.Background(), ev, uuid.New()); err != nil {
		t.Fatal(err)
	}
	if len(repo.created) != 0 {
		t.Fatalf("past event must not schedule, got %d batches", len(repo.created))
	}
}

func TestScheduleForEvent_SoonOnlyFuture(t *testing.T) {
	// event in 30m: -24h and -1h are past, only -15m is future
	repo := &mockRemindersRepo{}
	svc := NewRemindersService(repo, nil)
	ev := &events_domain.Event{ID: uuid.New(), StartAt: time.Now().UTC().Add(30 * time.Minute)}
	if err := svc.ScheduleForEvent(context.Background(), ev, uuid.New()); err != nil {
		t.Fatal(err)
	}
	if len(repo.created) != 1 || len(repo.created[0]) != 1 {
		t.Fatalf("expected 1 reminder, got %v", repo.created)
	}
}

func TestScheduleForTask_FutureAndPast(t *testing.T) {
	repo := &mockRemindersRepo{}
	svc := NewRemindersService(repo, nil)
	future := &tasks_domain.Task{ID: uuid.New(), DueAt: time.Now().UTC().Add(48 * time.Hour)}
	if err := svc.ScheduleForTask(context.Background(), future, uuid.New()); err != nil {
		t.Fatal(err)
	}
	if len(repo.created[0]) != 2 {
		t.Fatalf("future task needs 2 reminders, got %d", len(repo.created[0]))
	}
	repo2 := &mockRemindersRepo{}
	svc2 := NewRemindersService(repo2, nil)
	past := &tasks_domain.Task{ID: uuid.New(), DueAt: time.Now().UTC().Add(-time.Hour)}
	if err := svc2.ScheduleForTask(context.Background(), past, uuid.New()); err != nil {
		t.Fatal(err)
	}
	if len(repo2.created) != 0 {
		t.Fatalf("past task must not schedule")
	}
}

func TestUpdateReminders_DeletesThenSchedules(t *testing.T) {
	repo := &mockRemindersRepo{}
	svc := NewRemindersService(repo, nil)
	ev := &events_domain.Event{ID: uuid.New(), StartAt: time.Now().UTC().Add(48 * time.Hour)}
	if err := svc.UpdateRemindersForEvent(context.Background(), ev, uuid.New()); err != nil {
		t.Fatal(err)
	}
	if len(repo.deleted) != 1 || repo.deleted[0][0] != "event" {
		t.Fatalf("expected delete event reminders, got %v", repo.deleted)
	}
	if len(repo.created) != 1 {
		t.Fatalf("expected reschedule")
	}
	task := &tasks_domain.Task{ID: uuid.New(), DueAt: time.Now().UTC().Add(48 * time.Hour)}
	if err := svc.UpdateRemindersForTask(context.Background(), task, uuid.New()); err != nil {
		t.Fatal(err)
	}
	if len(repo.deleted) != 2 || repo.deleted[1][0] != "task" {
		t.Fatalf("expected delete task reminders, got %v", repo.deleted)
	}
}

func TestProcessPending_Empty(t *testing.T) {
	repo := &mockRemindersRepo{pending: nil}
	svc := NewRemindersService(repo, nil)
	n, err := svc.ProcessPendingReminders(context.Background(), time.Now(), 100)
	if err != nil || n != 0 {
		t.Fatalf("empty should be 0,nil got %d,%v", n, err)
	}
	if len(repo.marked) != 0 {
		t.Fatalf("nothing to mark")
	}
}

func TestProcessPending_MarksSent(t *testing.T) {
	repo := &mockRemindersRepo{pending: []*domain.Reminder{{ID: uuid.New()}, {ID: uuid.New()}}}
	svc := NewRemindersService(repo, nil)
	n, err := svc.ProcessPendingReminders(context.Background(), time.Now(), 100)
	if err != nil || n != 2 {
		t.Fatalf("expected 2, got %d,%v", n, err)
	}
	if len(repo.marked) != 1 || len(repo.marked[0]) != 2 {
		t.Fatalf("expected mark 2 ids, got %v", repo.marked)
	}
}

func TestProcessPending_RepoError(t *testing.T) {
	repo := &mockRemindersRepo{err: errors.New("db")}
	svc := NewRemindersService(repo, nil)
	if _, err := svc.ProcessPendingReminders(context.Background(), time.Now(), 10); err == nil {
		t.Fatalf("expected error")
	}
}

func TestDeleteReminders(t *testing.T) {
	repo := &mockRemindersRepo{}
	svc := NewRemindersService(repo, nil)
	id := uuid.New()
	if err := svc.DeleteRemindersForEvent(context.Background(), id); err != nil {
		t.Fatal(err)
	}
	if err := svc.DeleteRemindersForTask(context.Background(), id); err != nil {
		t.Fatal(err)
	}
	if len(repo.deleted) != 2 {
		t.Fatalf("expected 2 deletes")
	}
}

type fakeSender struct {
	sent   []string // endpoints
	fail   map[string]error
	called int
}

func (f *fakeSender) Send(ctx context.Context, sub *domain.PushSubscription, payload []byte) error {
	f.called++
	f.sent = append(f.sent, sub.Endpoint)
	if err, ok := f.fail[sub.Endpoint]; ok {
		return err
	}
	return nil
}

func TestProcessPending_SendsPushToEachDevice(t *testing.T) {
	uid := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	repo := &subsRepo{
		mockRemindersRepo: &mockRemindersRepo{
			pending: []*domain.Reminder{
				{ID: uuid.New(), UserID: uid, EntityType: "event", Title: "Созвон"},
				{ID: uuid.New(), UserID: uid, EntityType: "task", Title: "Отчёт"},
			},
		},
		subs: []*domain.PushSubscription{
			{ID: uuid.New(), Endpoint: "https://push/a"},
			{ID: uuid.New(), Endpoint: "https://push/b"},
		},
	}
	sender := &fakeSender{}
	svc := NewRemindersService(repo, sender)
	n, err := svc.ProcessPendingReminders(context.Background(), time.Now(), 100)
	if err != nil || n != 2 {
		t.Fatalf("expected 2,nil got %d,%v", n, err)
	}
	// 2 reminders x 2 devices
	if sender.called != 4 {
		t.Fatalf("expected 4 sends, got %d", sender.called)
	}
}

type subsRepo struct {
	*mockRemindersRepo
	subs    []*domain.PushSubscription
	deleted []string
}

func (s *subsRepo) GetSubscriptionsByUser(ctx context.Context, userID uuid.UUID) ([]*domain.PushSubscription, error) {
	return s.subs, nil
}

func (s *subsRepo) DeleteSubscriptionByEndpoint(ctx context.Context, userID uuid.UUID, endpoint string) error {
	s.deleted = append(s.deleted, endpoint)
	return nil
}

func TestProcessPending_GoneSubscriptionDeleted(t *testing.T) {
	uid := uuid.New()
	repo := &subsRepo{
		mockRemindersRepo: &mockRemindersRepo{
			pending: []*domain.Reminder{{ID: uuid.New(), UserID: uid, EntityType: "task", Title: "X"}},
		},
		subs: []*domain.PushSubscription{{ID: uuid.New(), Endpoint: "https://push/dead"}},
	}
	sender := &fakeSender{fail: map[string]error{"https://push/dead": ErrSubscriptionGone}}
	svc := NewRemindersService(repo, sender)
	n, err := svc.ProcessPendingReminders(context.Background(), time.Now(), 100)
	if err != nil || n != 1 {
		t.Fatalf("send failure must not fail batch: %d,%v", n, err)
	}
	if len(repo.deleted) != 1 || repo.deleted[0] != "https://push/dead" {
		t.Fatalf("dead endpoint must be deleted, got %v", repo.deleted)
	}
}

func TestProcessPending_NoSenderSkipsPush(t *testing.T) {
	repo := &mockRemindersRepo{pending: []*domain.Reminder{{ID: uuid.New()}}}
	svc := NewRemindersService(repo, nil)
	n, err := svc.ProcessPendingReminders(context.Background(), time.Now(), 100)
	if err != nil || n != 1 {
		t.Fatalf("expected 1,nil got %d,%v", n, err)
	}
}

func TestSaveDeletePushSubscription(t *testing.T) {
	repo := &mockRemindersRepo{}
	svc := NewRemindersService(repo, nil)
	uid := uuid.New()
	in := domain.PushSubscriptionInput{Endpoint: "https://push/x", P256DH: "p256dh-key-data", Auth: "auth-secret-data"}
	if err := svc.SavePushSubscription(context.Background(), uid, in); err != nil {
		t.Fatal(err)
	}
	if err := svc.DeletePushSubscription(context.Background(), uid, in.Endpoint); err != nil {
		t.Fatal(err)
	}
}

func TestBuildPushPayload(t *testing.T) {
	p := BuildPushPayload(&domain.Reminder{EntityType: "event", Title: "Созвон"})
	if string(p) == "" || !containsStr(string(p), "Созвон") {
		t.Fatalf("payload must contain title: %s", p)
	}
	p2 := BuildPushPayload(&domain.Reminder{EntityType: "task", Title: ""})
	if !containsStr(string(p2), "Новое напоминание") && !containsStr(string(p2), "Напоминание") {
		t.Fatalf("empty title must fall back: %s", p2)
	}
}

func containsStr(s, sub string) bool {
	return len(s) >= len(sub) && (func() bool {
		for i := 0; i+len(sub) <= len(s); i++ {
			if s[i:i+len(sub)] == sub {
				return true
			}
		}
		return false
	})()
}

type captureLogger struct {
	warns []string
	debug []string
}

func (l *captureLogger) Debug(msg string, fields ...zap.Field) { l.debug = append(l.debug, msg) }
func (l *captureLogger) Info(msg string, fields ...zap.Field)  {}
func (l *captureLogger) Warn(msg string, fields ...zap.Field)  { l.warns = append(l.warns, msg) }
func (l *captureLogger) Error(msg string, fields ...zap.Field) {}

func TestProcessPending_LogsDeliveryFailure(t *testing.T) {
	uid := uuid.New()
	repo := &subsRepo{
		mockRemindersRepo: &mockRemindersRepo{
			pending: []*domain.Reminder{{ID: uuid.New(), UserID: uid, EntityType: "task", Title: "X"}},
		},
		subs: []*domain.PushSubscription{{ID: uuid.New(), Endpoint: "https://push.example/broken"}},
	}
	sender := &fakeSender{fail: map[string]error{"https://push.example/broken": errors.New("connection refused")}}
	svc := NewRemindersService(repo, sender)
	log := &captureLogger{}
	svc.SetLogger(log)

	n, err := svc.ProcessPendingReminders(context.Background(), time.Now(), 100)
	if err != nil || n != 1 {
		t.Fatalf("failure must not fail batch: %d,%v", n, err)
	}
	if len(log.warns) != 1 || log.warns[0] != "push delivery failed" {
		t.Fatalf("expected one delivery-failure warn, got %v", log.warns)
	}
	// failed (non-gone) endpoint must be kept for retry
	if len(repo.deleted) != 0 {
		t.Fatalf("failed endpoint must be kept, got %v", repo.deleted)
	}
}

func TestProcessPending_NoLoggerNoPanic(t *testing.T) {
	uid := uuid.New()
	repo := &subsRepo{
		mockRemindersRepo: &mockRemindersRepo{
			pending: []*domain.Reminder{{ID: uuid.New(), UserID: uid, EntityType: "task", Title: "X"}},
		},
		subs: []*domain.PushSubscription{{ID: uuid.New(), Endpoint: "https://push.example/broken"}},
	}
	sender := &fakeSender{fail: map[string]error{"https://push.example/broken": errors.New("boom")}}
	svc := NewRemindersService(repo, sender) // no SetLogger call
	if _, err := svc.ProcessPendingReminders(context.Background(), time.Now(), 100); err != nil {
		t.Fatalf("nil logger must not panic or fail: %v", err)
	}
}

func TestNewWebPushSender_NormalizesSubject(t *testing.T) {
	// webpush-go prepends "mailto:" itself: passing "mailto:x@y" would
	// produce "mailto:mailto:x@y" and Apple answers 403 BadJwtToken.
	if got := NewWebPushSender("pub", "priv", "mailto:admin@example.com").subject; got != "admin@example.com" {
		t.Fatalf("mailto: prefix must be stripped, got %q", got)
	}
	if got := NewWebPushSender("pub", "priv", "admin@example.com").subject; got != "admin@example.com" {
		t.Fatalf("bare email must pass through, got %q", got)
	}
	if got := NewWebPushSender("pub", "priv", "https://example.com").subject; got != "https://example.com" {
		t.Fatalf("https URL must pass through, got %q", got)
	}
	if got := NewWebPushSender("pub", "priv", "").subject; got == "" || len(got) >= len("mailto:")+1 && got[:7] == "mailto:" {
		t.Fatalf("default subject must be bare (no mailto: prefix), got %q", got)
	}
}

func TestEndpointHost(t *testing.T) {
	if got := endpointHost("https://fcm.googleapis.com/fcm/send/abc"); got != "fcm.googleapis.com" {
		t.Fatalf("got %q", got)
	}
	if got := endpointHost("not a url \\"); got != "unknown" {
		t.Fatalf("got %q", got)
	}
}
