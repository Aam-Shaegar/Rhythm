package delete

import (
	"context"
	"net/http"

	core_errors "github.com/Aam-Shaegar/Rhythm/internal/core/errors"
	core_response "github.com/Aam-Shaegar/Rhythm/internal/core/transport/http/response"
	"github.com/Aam-Shaegar/Rhythm/internal/features/events/service"
	"github.com/google/uuid"
)

type Handler struct {
	svc service.EventsService
}

func NewHandler(svc service.EventsService) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	userIDStr, ok := getUserIDFromContext(ctx)
	if !ok {
		core_response.NewHTTPResponseHandler(nil, w).ErrorResponse(core_errors.ErrUnauthorized, "user not in context")
		return
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		core_response.NewHTTPResponseHandler(nil, w).ErrorResponse(core_errors.ErrInvalidArgument, "invalid user id")
		return
	}

	eventIDStr := r.PathValue("id")
	eventID, err := uuid.Parse(eventIDStr)
	if err != nil {
		core_response.NewHTTPResponseHandler(nil, w).ErrorResponse(core_errors.ErrInvalidArgument, "invalid event id")
		return
	}

	if err := h.svc.DeleteEvent(ctx, userID, eventID); err != nil {
		core_response.NewHTTPResponseHandler(nil, w).ErrorResponse(err, "failed to delete event")
		return
	}

	respHandler := core_response.NewHTTPResponseHandler(nil, w)
	respHandler.NoContentResponse()
}

func getUserIDFromContext(ctx context.Context) (string, bool) {
	val := ctx.Value("user_id")
	if val == nil {
		return "", false
	}
	id, ok := val.(string)
	return id, ok
}