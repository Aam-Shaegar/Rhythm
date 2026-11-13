package http

import (
	"context"
	core_errors "github.com/Aam-Shaegar/Rhythm/internal/core/errors"
	middleware "github.com/Aam-Shaegar/Rhythm/internal/core/transport/http/middleware"
	response "github.com/Aam-Shaegar/Rhythm/internal/core/transport/http/response"
	server "github.com/Aam-Shaegar/Rhythm/internal/core/transport/http/server"
	"github.com/Aam-Shaegar/Rhythm/internal/features/priorities/domain"
	"github.com/google/uuid"
	"net/http"
)

type Service interface {
	SyncUser(context.Context, uuid.UUID) (*domain.Response, error)
}
type Handler struct{ svc Service }

func New(s Service) *Handler { return &Handler{svc: s} }
func (h *Handler) Routes() []server.Route {
	return []server.Route{server.NewRoute(http.MethodGet, "/tasks/priorities", h.Get)}
}
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	out := response.NewHTTPResponseHandler(nil, w)
	id, ok := middleware.GetUserID(r.Context())
	if !ok {
		out.ErrorResponse(core_errors.ErrUnauthorized, "authentication required")
		return
	}
	uid, err := uuid.Parse(id)
	if err != nil {
		out.ErrorResponse(core_errors.ErrUnauthorized, "invalid user")
		return
	}
	result, err := h.svc.SyncUser(r.Context(), uid)
	if err != nil {
		out.ErrorResponse(err, "Не удалось рассчитать приоритеты. Представление по времени доступно.")
		return
	}
	out.JSONResponse(result, http.StatusOK)
}
