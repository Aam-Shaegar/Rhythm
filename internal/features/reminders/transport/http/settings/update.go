package settings

import (
	"net/http"

	core_errors "github.com/Aam-Shaegar/Rhythm/internal/core/errors"
	core_request "github.com/Aam-Shaegar/Rhythm/internal/core/transport/http/request"
	core_response "github.com/Aam-Shaegar/Rhythm/internal/core/transport/http/response"
	"github.com/Aam-Shaegar/Rhythm/internal/features/reminders/domain"
	"github.com/Aam-Shaegar/Rhythm/internal/features/reminders/service"
)

type UpdateHandler struct {
	svc service.RemindersService
}

func NewUpdateHandler(svc service.RemindersService) *UpdateHandler {
	return &UpdateHandler{svc: svc}
}

func (h *UpdateHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	_, ok := getUserIDFromContext(ctx)
	if !ok {
		core_response.NewHTTPResponseHandler(nil, w).ErrorResponse(core_errors.ErrUnauthorized, "user not in context")
		return
	}

	var input domain.UpdateReminderSettingsInput
	if err := core_request.DecodeAndValidateRequest(r, &input); err != nil {
		core_response.NewHTTPResponseHandler(nil, w).ErrorResponse(err, "invalid request")
		return
	}

	// In a full implementation, you'd save to a user settings table
	// For now, just return the updated settings
	settings := domain.ReminderSettings{
		EventReminders:  derefBool(input.EventReminders, true),
		TaskReminders:   derefBool(input.TaskReminders, true),
		EventBefore15m:  derefBool(input.EventBefore15m, true),
		EventBefore1h:   derefBool(input.EventBefore1h, true),
		EventBefore24h:  derefBool(input.EventBefore24h, true),
		TaskBefore1h:    derefBool(input.TaskBefore1h, true),
		TaskBefore24h:   derefBool(input.TaskBefore24h, true),
	}

	respHandler := core_response.NewHTTPResponseHandler(nil, w)
	respHandler.JSONResponse(settings, http.StatusOK)
}

func derefBool(ptr *bool, defaultVal bool) bool {
	if ptr != nil {
		return *ptr
	}
	return defaultVal
}