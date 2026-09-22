package daily

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

	var query domain.DailyReportQuery
	if err := core_request.DecodeQueryParams(r, &query); err != nil {
		core_response.NewHTTPResponseHandler(nil, w).ErrorResponse(err, "invalid query params")
		return
	}

	date := query.Date
	if date.IsZero() {
		date = time.Now().UTC()
	}

	report, err := h.svc.GetDailyReport(ctx, userID, date)
	if err != nil {
		core_response.NewHTTPResponseHandler(nil, w).ErrorResponse(err, "failed to get daily report")
		return
	}

	respHandler := core_response.NewHTTPResponseHandler(nil, w)
	respHandler.JSONResponse(report, http.StatusOK)
}

func getUserIDFromContext(ctx context.Context) (string, bool) {
	val := ctx.Value("user_id")
	if val == nil {
		return "", false
	}
	id, ok := val.(string)
	return id, ok
}
