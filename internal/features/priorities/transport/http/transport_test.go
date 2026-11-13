package http

import (
	"context"
	logger "github.com/Aam-Shaegar/Rhythm/internal/core/logger"
	middleware "github.com/Aam-Shaegar/Rhythm/internal/core/transport/http/middleware"
	"github.com/Aam-Shaegar/Rhythm/internal/features/priorities/domain"
	"github.com/google/uuid"
	"net/http"
	"net/http/httptest"
	"testing"
)

type fakeService struct {
	got   uuid.UUID
	calls int
}

func (s *fakeService) SyncUser(_ context.Context, id uuid.UUID) (*domain.Response, error) {
	s.got = id
	s.calls++
	return &domain.Response{ModelVersion: "test", Items: []domain.Item{}}, nil
}
func TestAuthenticationOwnsUserID(t *testing.T) {
	svc := &fakeService{}
	h := New(svc)
	request := httptest.NewRequest("GET", "/tasks/priorities?user_id="+uuid.NewString(), nil)
	w := httptest.NewRecorder()
	h.Get(w, request)
	if w.Code != 401 || svc.calls != 0 {
		t.Fatal("anonymous access allowed")
	}
	own := uuid.New()
	wrapped := middleware.Auth(func(string) (string, string, error) { return own.String(), "user", nil })(http.HandlerFunc(h.Get))
	l, err := logger.NewLogger(logger.LoggerConfig{Folder: t.TempDir(), Level: "ERROR"})
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()
	request = request.WithContext(logger.ToContext(request.Context(), l))
	request.Header.Set("Authorization", "Bearer test")
	w = httptest.NewRecorder()
	wrapped.ServeHTTP(w, request)
	if w.Code != 200 || svc.got != own || svc.calls != 1 {
		t.Fatal("query user_id replaced authenticated identity")
	}
}
