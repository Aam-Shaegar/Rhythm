package get_user

import (
	"net/http"

	core_errors "github.com/Aam-Shaegar/Rhythm/internal/core/errors"
	core_middleware "github.com/Aam-Shaegar/Rhythm/internal/core/transport/http/middleware"
	core_response "github.com/Aam-Shaegar/Rhythm/internal/core/transport/http/response"
	"github.com/Aam-Shaegar/Rhythm/internal/features/users/service"
	"github.com/google/uuid"
)

type Handler struct {
	svc service.UsersService
}

func NewHandler(svc service.UsersService) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	userIDStr, ok := core_middleware.GetUserID(ctx)
	if !ok {
		core_response.NewHTTPResponseHandler(nil, w).ErrorResponse(core_errors.ErrUnauthorized, "user not in context")
		return
	}

	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		core_response.NewHTTPResponseHandler(nil, w).ErrorResponse(core_errors.ErrInvalidArgument, "invalid user id")
		return
	}

	user, err := h.svc.GetMe(ctx, userID)
	if err != nil {
		core_response.NewHTTPResponseHandler(nil, w).ErrorResponse(err, "failed to get user")
		return
	}

	respHandler := core_response.NewHTTPResponseHandler(nil, w)
	respHandler.JSONResponse(user, http.StatusOK)
}
