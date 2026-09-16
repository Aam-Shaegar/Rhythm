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

func TestCreateEvent_InvalidFormat(t *testing.T) {
	svc := NewEventsService(newMockEventsRepo(), newMockRemindersService())
	for _, tc := range []struct{ start, end string }{
		{"bad", "2026-09-20T11:00:00Z"},
		{"2026-09-20T10:00:00Z", "bad"},
		{"2026-09-20", "2026-09-21"},
		{"", ""},
	} {
		_, err := svc.CreateEvent(context.Background(), uuid.New(), domain.CreateEventInput{
			Title: "t", StartAt: tc.start, EndAt: tc.end,
		})
		if !errors.Is(err, core_errors.ErrInvalidArgument) {
			t.Fatalf("start=%q end=%q: expected InvalidArgument got %v", tc.start, tc.end, err)
		}
	}
}

func TestCreateEvent_EndEqualsStart(t *testing.T) {
	svc := NewEventsService(newMockEventsRepo(), newMockRemindersService())
	ts := "2026-09-20T10:00:00Z"
	_, err := svc.CreateEvent(context.Background(), uuid.New(), domain.CreateEventInput{Title: "t", StartAt: ts, EndAt: ts})
	if !errors.Is(err, core_errors.ErrInvalidArgument) {
		t.Fatalf("equal start/end should be invalid, got %v", err)
	}
}

func TestUpdateEvent_EndBeforeStart(t *testing.T) {
	repo := newMockEventsRepo()
	svc := NewEventsService(repo, newMockRemindersService())
	uid := uuid.New()
	ev, _ := svc.CreateEvent(context.Background(), uid, domain.CreateEventInput{
		Title: "t", StartAt: "2026-09-20T10:00:00Z", EndAt: "2026-09-20T11:00:00Z",
	})
	id, _ := uuid.Parse(ev.ID.String())
	badEnd := "2026-09-20T09:00:00Z"
	_, err := svc.UpdateEvent(context.Background(), uid, id, domain.UpdateEventInput{EndAt: &badEnd})
	if !errors.Is(err, core_errors.ErrInvalidArgument) {
		t.Fatalf("expected invalid end, got %v", err)
	}
}

func TestUpdateEvent_InvalidDateFormat(t *testing.T) {
	repo := newMockEventsRepo()
	svc := NewEventsService(repo, newMockRemindersService())
	uid := uuid.New()
	ev, _ := svc.CreateEvent(context.Background(), uid, domain.CreateEventInput{
		Title: "t", StartAt: "2026-09-20T10:00:00Z", EndAt: "2026-09-20T11:00:00Z",
	})
	id, _ := uuid.Parse(ev.ID.String())
	bad := "tomorrow"
	_, err := svc.UpdateEvent(context.Background(), uid, id, domain.UpdateEventInput{StartAt: &bad})
	if !errors.Is(err, core_errors.ErrInvalidArgument) {
		t.Fatalf("expected invalid start, got %v", err)
	}
}

func TestUpdateEvent_Edge_WrongUser(t *testing.T) {
	repo := newMockEventsRepo()
	svc := NewEventsService(repo, newMockRemindersService())
	uid := uuid.New()
	ev, _ := svc.CreateEvent(context.Background(), uid, domain.CreateEventInput{
		Title: "t", StartAt: "2026-09-20T10:00:00Z", EndAt: "2026-09-20T11:00:00Z",
	})
	id, _ := uuid.Parse(ev.ID.String())
	nt := "hacked"
	_, err := svc.UpdateEvent(context.Background(), uuid.New(), id, domain.UpdateEventInput{Title: &nt})
	if !errors.Is(err, core_errors.ErrNotFound) {
		t.Fatalf("expected NotFound, got %v", err)
	}
}

func TestGetEvent_Edge_WrongUser(t *testing.T) {
	repo := newMockEventsRepo()
	svc := NewEventsService(repo, newMockRemindersService())
	uid := uuid.New()
	ev, _ := svc.CreateEvent(context.Background(), uid, domain.CreateEventInput{
		Title: "t", StartAt: "2026-09-20T10:00:00Z", EndAt: "2026-09-20T11:00:00Z",
	})
	id, _ := uuid.Parse(ev.ID.String())
	_, err := svc.GetEvent(context.Background(), uuid.New(), id)
	if !errors.Is(err, core_errors.ErrNotFound) {
		t.Fatalf("expected NotFound, got %v", err)
	}
}

func TestListEvents_Empty(t *testing.T) {
	svc := NewEventsService(newMockEventsRepo(), newMockRemindersService())
	list, err := svc.ListEvents(context.Background(), uuid.New(), domain.EventFilter{})
	if err != nil || len(list) != 0 {
		t.Fatalf("expected empty, got %v %v", list, err)
	}
}

func TestDeleteEvent_Edge_WrongUser(t *testing.T) {
	repo := newMockEventsRepo()
	svc := NewEventsService(repo, newMockRemindersService())
	uid := uuid.New()
	ev, _ := svc.CreateEvent(context.Background(), uid, domain.CreateEventInput{
		Title: "t", StartAt: time.Now().Add(time.Hour).Format(time.RFC3339), EndAt: time.Now().Add(2 * time.Hour).Format(time.RFC3339),
	})
	id, _ := uuid.Parse(ev.ID.String())
	if err := svc.DeleteEvent(context.Background(), uuid.New(), id); !errors.Is(err, core_errors.ErrNotFound) {
		t.Fatalf("expected NotFound, got %v", err)
	}
}

func TestDeleteEvent_NotFound(t *testing.T) {
	svc := NewEventsService(newMockEventsRepo(), newMockRemindersService())
	if err := svc.DeleteEvent(context.Background(), uuid.New(), uuid.New()); !errors.Is(err, core_errors.ErrNotFound) {
		t.Fatalf("expected NotFound, got %v", err)
	}
}
