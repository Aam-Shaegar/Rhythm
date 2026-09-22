package settings

import (
	"context"
	"net/http"

	core_errors "github.com/Aam-Shaegar/Rhythm/internal/core/errors"
	core_response "github.com/Aam-Shaegar/Rhythm/internal/core/transport/http/response"
	"github.com/Aam-Shaegar/Rhythm/internal/features/reminders/domain"
	"github.com/Aam-Shaegar/Rhythm/internal/features/reminders/service"
)

type Handler struct {
	svc service.RemindersService
}

func NewHandler(svc service.RemindersService) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	_, ok := getUserIDFromContext(ctx)
	if !ok {
		core_response.NewHTTPResponseHandler(nil, w).ErrorResponse(core_errors.ErrUnauthorized, "user not in context")
		return
	}

	settings := domain.ReminderSettings{
		EventReminders: true,
		TaskReminders:  true,
		EventBefore15m: true,
		EventBefore1h:  true,
		EventBefore24h: true,
		TaskBefore1h:   true,
		TaskBefore24h:  true,
	}

	respHandler := core_response.NewHTTPResponseHandler(nil, w)
	respHandler.JSONResponse(settings, http.StatusOK)
}

func getUserIDFromContext(ctx context.Context) (string, bool) {
	val := ctx.Value("user_id")
	if val == nil {
		return "", false
	}
	id, ok := val.(string)
	return id, ok
}
