package service

import (
	"context"
	"errors"
	"testing"
	"time"

	core_errors "github.com/Aam-Shaegar/Rhythm/internal/core/errors"
	"github.com/Aam-Shaegar/Rhythm/internal/features/events/domain"
	"github.com/google/uuid"
)

type mockEventsRepo struct {
	events map[uuid.UUID]*domain.Event
	byUser map[uuid.UUID][]*domain.Event
}

func newMockEventsRepo() *mockEventsRepo {
	return &mockEventsRepo{
		events: make(map[uuid.UUID]*domain.Event),
		byUser: make(map[uuid.UUID][]*domain.Event),
	}
}

func (m *mockEventsRepo) Create(ctx context.Context, event *domain.Event) error {
	if _, exists := m.events[event.ID]; exists {
		return core_errors.ErrConflict
	}
	m.events[event.ID] = event
	m.byUser[event.UserID] = append(m.byUser[event.UserID], event)
	return nil
}

func (m *mockEventsRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.Event, error) {
	if e, ok := m.events[id]; ok {
		return e, nil
	}
	return nil, core_errors.ErrNotFound
}

func (m *mockEventsRepo) GetByUserID(ctx context.Context, filter domain.EventFilter) ([]*domain.Event, error) {
	userEvents, ok := m.byUser[filter.UserID]
	if !ok {
		return []*domain.Event{}, nil
	}

	var result []*domain.Event
	for _, e := range userEvents {
		if filter.DateFrom != nil && e.StartAt.Before(*filter.DateFrom) {
			continue
		}
		if filter.DateTo != nil && e.StartAt.After(*filter.DateTo) {
			continue
		}
		result = append(result, e)
	}
	return result, nil
}

func (m *mockEventsRepo) Update(ctx context.Context, event *domain.Event) error {
	if _, ok := m.events[event.ID]; !ok {
		return core_errors.ErrNotFound
	}
	m.events[event.ID] = event

	// Update in byUser
	for i, e := range m.byUser[event.UserID] {
		if e.ID == event.ID {
			m.byUser[event.UserID][i] = event
			break
		}
	}
	return nil
}

func (m *mockEventsRepo) Delete(ctx context.Context, id uuid.UUID) error {
	event, ok := m.events[id]
	if !ok {
		return core_errors.ErrNotFound
	}
	delete(m.events, id)

	// Remove from byUser
	userEvents := m.byUser[event.UserID]
	for i, e := range userEvents {
		if e.ID == id {
			m.byUser[event.UserID] = append(userEvents[:i], userEvents[i+1:]...)
			break
		}
	}
	return nil
}

func (m *mockEventsRepo) GetUpcoming(ctx context.Context, userID uuid.UUID, before time.Time, limit int) ([]*domain.Event, error) {
	userEvents, ok := m.byUser[userID]
	if !ok {
		return []*domain.Event{}, nil
	}

	var result []*domain.Event
	for _, e := range userEvents {
		if e.StartAt.Before(before) || e.StartAt.Equal(before) {
			result = append(result, e)
			if len(result) >= limit {
				break
			}
		}
	}
	return result, nil
}

type mockRemindersService struct {
	scheduledEvents []*domain.Event
	updatedEvents   []*domain.Event
	deletedEvents   []uuid.UUID
}

func newMockRemindersService() *mockRemindersService {
	return &mockRemindersService{
		scheduledEvents: make([]*domain.Event, 0),
		updatedEvents:   make([]*domain.Event, 0),
		deletedEvents:   make([]uuid.UUID, 0),
	}
}

func (m *mockRemindersService) ScheduleForEvent(ctx context.Context, event *domain.Event, userID uuid.UUID) error {
	m.scheduledEvents = append(m.scheduledEvents, event)
	return nil
}

func (m *mockRemindersService) UpdateRemindersForEvent(ctx context.Context, event *domain.Event, userID uuid.UUID) error {
	m.updatedEvents = append(m.updatedEvents, event)
	return nil
}

func (m *mockRemindersService) DeleteRemindersForEvent(ctx context.Context, eventID uuid.UUID) error {
	m.deletedEvents = append(m.deletedEvents, eventID)
	return nil
}

func TestCreateEvent_Success(t *testing.T) {
	repo := newMockEventsRepo()
	reminders := newMockRemindersService()
	svc := NewEventsService(repo, reminders)

	userID := uuid.New()
	input := domain.CreateEventInput{
		Title:       "Test Event",
		Description: strPtr("Test Description"),
		StartAt:     time.Now().Add(time.Hour).Format(time.RFC3339),
		EndAt:       time.Now().Add(2 * time.Hour).Format(time.RFC3339),
	}

	event, err := svc.CreateEvent(context.Background(), userID, input)
	if err != nil {
		t.Fatalf("CreateEvent failed: %v", err)
	}

	if event.Title != "Test Event" {
		t.Errorf("expected title 'Test Event', got %s", event.Title)
	}
	if event.Description == nil || *event.Description != "Test Description" {
		t.Errorf("expected description 'Test Description', got %v", event.Description)
	}
}

func TestCreateEvent_InvalidTimeFormat(t *testing.T) {
	repo := newMockEventsRepo()
	reminders := newMockRemindersService()
	svc := NewEventsService(repo, reminders)

	userID := uuid.New()
	input := domain.CreateEventInput{
		Title:   "Test Event",
		StartAt: "invalid-time",
		EndAt:   time.Now().Add(time.Hour).Format(time.RFC3339),
	}

	_, err := svc.CreateEvent(context.Background(), userID, input)
	if err == nil {
		t.Fatal("expected error for invalid time format")
	}
	if !errors.Is(err, core_errors.ErrInvalidArgument) {
		t.Errorf("expected ErrInvalidArgument, got %v", err)
	}
}

func TestCreateEvent_EndBeforeStart(t *testing.T) {
	repo := newMockEventsRepo()
	reminders := newMockRemindersService()
	svc := NewEventsService(repo, reminders)

	userID := uuid.New()
	input := domain.CreateEventInput{
		Title:   "Test Event",
		StartAt: time.Now().Add(2 * time.Hour).Format(time.RFC3339),
		EndAt:   time.Now().Add(time.Hour).Format(time.RFC3339),
	}

	_, err := svc.CreateEvent(context.Background(), userID, input)
	if err == nil {
		t.Fatal("expected error for end before start")
	}
	if !errors.Is(err, core_errors.ErrInvalidArgument) {
		t.Errorf("expected ErrInvalidArgument, got %v", err)
	}
}

func TestGetEvent_Success(t *testing.T) {
	repo := newMockEventsRepo()
	reminders := newMockRemindersService()
	svc := NewEventsService(repo, reminders)

	userID := uuid.New()
	eventID := uuid.New()
	event := &domain.Event{
		ID:          eventID,
		UserID:      userID,
		Title:       "Test Event",
		Description: strPtr("Test Description"),
		StartAt:     time.Now().Add(time.Hour),
		EndAt:       time.Now().Add(2 * time.Hour),
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	repo.Create(context.Background(), event)

	result, err := svc.GetEvent(context.Background(), userID, eventID)
	if err != nil {
		t.Fatalf("GetEvent failed: %v", err)
	}

	if result.ID != eventID {
		t.Errorf("expected event ID %v, got %v", eventID, result.ID)
	}
	if result.Title != "Test Event" {
		t.Errorf("expected title 'Test Event', got %s", result.Title)
	}
}

func TestGetEvent_NotFound(t *testing.T) {
	repo := newMockEventsRepo()
	reminders := newMockRemindersService()
	svc := NewEventsService(repo, reminders)

	_, err := svc.GetEvent(context.Background(), uuid.New(), uuid.New())
	if err == nil {
		t.Fatal("expected not found error")
	}
	if !errors.Is(err, core_errors.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestGetEvent_WrongUser(t *testing.T) {
	repo := newMockEventsRepo()
	reminders := newMockRemindersService()
	svc := NewEventsService(repo, reminders)

	userID := uuid.New()
	otherUserID := uuid.New()
	eventID := uuid.New()
	event := &domain.Event{
		ID:          eventID,
		UserID:      userID,
		Title:       "Test Event",
		StartAt:     time.Now().Add(time.Hour),
		EndAt:       time.Now().Add(2 * time.Hour),
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	repo.Create(context.Background(), event)

	_, err := svc.GetEvent(context.Background(), otherUserID, eventID)
	if err == nil {
		t.Fatal("expected not found error for other user")
	}
	if !errors.Is(err, core_errors.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestListEvents_Success(t *testing.T) {
	repo := newMockEventsRepo()
	reminders := newMockRemindersService()
	svc := NewEventsService(repo, reminders)

	userID := uuid.New()
	now := time.Now()

	event1 := &domain.Event{
		ID:          uuid.New(),
		UserID:      userID,
		Title:       "Event 1",
		StartAt:     now.Add(time.Hour),
		EndAt:       now.Add(2 * time.Hour),
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	event2 := &domain.Event{
		ID:          uuid.New(),
		UserID:      userID,
		Title:       "Event 2",
		StartAt:     now.Add(3 * time.Hour),
		EndAt:       now.Add(4 * time.Hour),
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	repo.Create(context.Background(), event1)
	repo.Create(context.Background(), event2)

	filter := domain.EventFilter{}
	events, err := svc.ListEvents(context.Background(), userID, filter)
	if err != nil {
		t.Fatalf("ListEvents failed: %v", err)
	}

	if len(events) != 2 {
		t.Errorf("expected 2 events, got %d", len(events))
	}
}

func TestListEvents_WithDateFilter(t *testing.T) {
	repo := newMockEventsRepo()
	reminders := newMockRemindersService()
	svc := NewEventsService(repo, reminders)

	userID := uuid.New()
	now := time.Now()

	event1 := &domain.Event{
		ID:          uuid.New(),
		UserID:      userID,
		Title:       "Event 1",
		StartAt:     now.Add(time.Hour),
		EndAt:       now.Add(2 * time.Hour),
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	event2 := &domain.Event{
		ID:          uuid.New(),
		UserID:      userID,
		Title:       "Event 2",
		StartAt:     now.Add(24 * time.Hour),
		EndAt:       now.Add(25 * time.Hour),
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	repo.Create(context.Background(), event1)
	repo.Create(context.Background(), event2)

	dateFrom := now.Add(12 * time.Hour)
	filter := domain.EventFilter{
		DateFrom: &dateFrom,
	}
	events, err := svc.ListEvents(context.Background(), userID, filter)
	if err != nil {
		t.Fatalf("ListEvents failed: %v", err)
	}

	if len(events) != 1 {
		t.Errorf("expected 1 event after date filter, got %d", len(events))
	}
	if events[0].Title != "Event 2" {
		t.Errorf("expected 'Event 2', got %s", events[0].Title)
	}
}

func TestUpdateEvent_Success(t *testing.T) {
	repo := newMockEventsRepo()
	reminders := newMockRemindersService()
	svc := NewEventsService(repo, reminders)

	userID := uuid.New()
	eventID := uuid.New()
	event := &domain.Event{
		ID:          eventID,
		UserID:      userID,
		Title:       "Original Title",
		Description: strPtr("Original Description"),
		StartAt:     time.Now().Add(time.Hour),
		EndAt:       time.Now().Add(2 * time.Hour),
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	repo.Create(context.Background(), event)

	newTitle := "Updated Title"
	newDesc := "Updated Description"
	newStart := time.Now().Add(3 * time.Hour).Format(time.RFC3339)
	newEnd := time.Now().Add(4 * time.Hour).Format(time.RFC3339)

	input := domain.UpdateEventInput{
		Title:       &newTitle,
		Description: &newDesc,
		StartAt:     &newStart,
		EndAt:       &newEnd,
	}

	result, err := svc.UpdateEvent(context.Background(), userID, eventID, input)
	if err != nil {
		t.Fatalf("UpdateEvent failed: %v", err)
	}

	if result.Title != "Updated Title" {
		t.Errorf("expected title 'Updated Title', got %s", result.Title)
	}
	if result.Description == nil || *result.Description != "Updated Description" {
		t.Errorf("expected description 'Updated Description', got %v", result.Description)
	}
}

func TestUpdateEvent_WrongUser(t *testing.T) {
	repo := newMockEventsRepo()
	reminders := newMockRemindersService()
	svc := NewEventsService(repo, reminders)

	userID := uuid.New()
	otherUserID := uuid.New()
	eventID := uuid.New()
	event := &domain.Event{
		ID:          eventID,
		UserID:      userID,
		Title:       "Test Event",
		StartAt:     time.Now().Add(time.Hour),
		EndAt:       time.Now().Add(2 * time.Hour),
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	repo.Create(context.Background(), event)

	input := domain.UpdateEventInput{
		Title: strPtr("New Title"),
	}

	_, err := svc.UpdateEvent(context.Background(), otherUserID, eventID, input)
	if err == nil {
		t.Fatal("expected not found error for other user")
	}
	if !errors.Is(err, core_errors.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestDeleteEvent_Success(t *testing.T) {
	repo := newMockEventsRepo()
	reminders := newMockRemindersService()
	svc := NewEventsService(repo, reminders)

	userID := uuid.New()
	eventID := uuid.New()
	event := &domain.Event{
		ID:          eventID,
		UserID:      userID,
		Title:       "Test Event",
		StartAt:     time.Now().Add(time.Hour),
		EndAt:       time.Now().Add(2 * time.Hour),
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	repo.Create(context.Background(), event)

	err := svc.DeleteEvent(context.Background(), userID, eventID)
	if err != nil {
		t.Fatalf("DeleteEvent failed: %v", err)
	}

	// Verify deleted
	_, err = svc.GetEvent(context.Background(), userID, eventID)
	if err == nil {
		t.Fatal("expected not found after delete")
	}
	if !errors.Is(err, core_errors.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestDeleteEvent_WrongUser(t *testing.T) {
	repo := newMockEventsRepo()
	reminders := newMockRemindersService()
	svc := NewEventsService(repo, reminders)

	userID := uuid.New()
	otherUserID := uuid.New()
	eventID := uuid.New()
	event := &domain.Event{
		ID:          eventID,
		UserID:      userID,
		Title:       "Test Event",
		StartAt:     time.Now().Add(time.Hour),
		EndAt:       time.Now().Add(2 * time.Hour),
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	repo.Create(context.Background(), event)

	err := svc.DeleteEvent(context.Background(), otherUserID, eventID)
	if err == nil {
		t.Fatal("expected not found error for other user")
	}
	if !errors.Is(err, core_errors.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func strPtr(s string) *string {
	return &s
}