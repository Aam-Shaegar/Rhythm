package settings

import (
	"context"
	"github.com/google/uuid"
	"net/http"

	core_errors "github.com/Aam-Shaegar/Rhythm/internal/core/errors"
	core_response "github.com/Aam-Shaegar/Rhythm/internal/core/transport/http/response"
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

	uid, ok := getUserIDFromContext(ctx)
	if !ok {
		core_response.NewHTTPResponseHandler(nil, w).ErrorResponse(core_errors.ErrUnauthorized, "user not in context")
		return
	}

	userID, err := uuid.Parse(uid)
	if err != nil {
		core_response.NewHTTPResponseHandler(nil, w).ErrorResponse(core_errors.ErrUnauthorized, "invalid user")
		return
	}
	settings, err := h.svc.GetSettings(ctx, userID)
	if err != nil {
		core_response.NewHTTPResponseHandler(nil, w).ErrorResponse(err, "не удалось сохранить настройки напоминаний")
		return
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
