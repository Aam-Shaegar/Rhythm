package settings

import (
	"github.com/google/uuid"
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

	uid, ok := getUserIDFromContext(ctx)
	if !ok {
		core_response.NewHTTPResponseHandler(nil, w).ErrorResponse(core_errors.ErrUnauthorized, "user not in context")
		return
	}

	var input domain.UpdateReminderSettingsInput
	if err := core_request.DecodeAndValidateRequest(r, &input); err != nil {
		core_response.NewHTTPResponseHandler(nil, w).ErrorResponse(err, "invalid request")
		return
	}

	userID, err := uuid.Parse(uid)
	if err != nil {
		core_response.NewHTTPResponseHandler(nil, w).ErrorResponse(core_errors.ErrUnauthorized, "invalid user")
		return
	}
	settings, err := h.svc.UpdateSettings(ctx, userID, input)
	if err != nil {
		core_response.NewHTTPResponseHandler(nil, w).ErrorResponse(err, "не удалось сохранить настройки напоминаний")
		return
	}

	respHandler := core_response.NewHTTPResponseHandler(nil, w)
	respHandler.JSONResponse(settings, http.StatusOK)
}
