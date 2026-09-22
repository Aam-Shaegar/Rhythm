package register

import (
	"net/http"

	core_request "github.com/Aam-Shaegar/Rhythm/internal/core/transport/http/request"
	core_response "github.com/Aam-Shaegar/Rhythm/internal/core/transport/http/response"
	"github.com/Aam-Shaegar/Rhythm/internal/features/users/domain/dtos"
	"github.com/Aam-Shaegar/Rhythm/internal/features/users/service"
)

type Handler struct {
	svc service.UsersService
}

func NewHandler(svc service.UsersService) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var input dtos.RegisterInput
	if err := core_request.DecodeAndValidateRequest(r, &input); err != nil {
		core_response.NewHTTPResponseHandler(nil, w).ErrorResponse(err, "invalid request")
		return
	}

	authResp, err := h.svc.Register(ctx, input)
	if err != nil {
		core_response.NewHTTPResponseHandler(nil, w).ErrorResponse(err, "registration failed")
		return
	}

	respHandler := core_response.NewHTTPResponseHandler(nil, w)
	respHandler.JSONResponse(authResp, http.StatusCreated)
}
