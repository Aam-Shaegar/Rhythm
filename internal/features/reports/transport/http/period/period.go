package period

import (
	"context"
	"net/http"
	"time"

	core_errors "github.com/Aam-Shaegar/Rhythm/internal/core/errors"
	core_request "github.com/Aam-Shaegar/Rhythm/internal/core/transport/http/request"
	core_response "github.com/Aam-Shaegar/Rhythm/internal/core/transport/http/response"
	"github.com/Aam-Shaegar/Rhythm/internal/features/reports/domain"
	"github.com/Aam-Shaegar/Rhythm/internal/features/reports/service"
	"github.com/google/uuid"
)

type Handler struct {
	svc service.ReportsService
}

func NewHandler(svc service.ReportsService) *Handler {
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

	var query domain.PeriodReportQuery
	if err := core_request.DecodeQueryParams(r, &query); err != nil {
		core_response.NewHTTPResponseHandler(nil, w).ErrorResponse(err, "invalid query params")
		return
	}

	from := query.From
	to := query.To

	if from.IsZero() {
		from = time.Now().UTC().AddDate(0, 0, -30)
	}
	if to.IsZero() {
		to = time.Now().UTC()
	}

	reports, err := h.svc.GetPeriodReport(ctx, userID, from, to)
	if err != nil {
		core_response.NewHTTPResponseHandler(nil, w).ErrorResponse(err, "failed to get period report")
		return
	}

	respHandler := core_response.NewHTTPResponseHandler(nil, w)
	respHandler.JSONResponse(reports, http.StatusOK)
}

func getUserIDFromContext(ctx context.Context) (string, bool) {
	val := ctx.Value("user_id")
	if val == nil {
		return "", false
	}
	id, ok := val.(string)
	return id, ok
}