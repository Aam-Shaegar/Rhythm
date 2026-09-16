package http

import (
	"net/http"

	core_http_server "github.com/Aam-Shaegar/Rhythm/internal/core/transport/http/server"
	"github.com/Aam-Shaegar/Rhythm/internal/features/reports/service"
	"github.com/Aam-Shaegar/Rhythm/internal/features/reports/transport/http/daily"
	"github.com/Aam-Shaegar/Rhythm/internal/features/reports/transport/http/period"
)

type ReportsHTTPHandler struct {
	svc service.ReportsService
}

func NewReportsHTTPHandler(svc service.ReportsService) *ReportsHTTPHandler {
	return &ReportsHTTPHandler{svc: svc}
}

func (h *ReportsHTTPHandler) Routes() []core_http_server.Route {
	return []core_http_server.Route{
		core_http_server.NewRoute(http.MethodGet, "/reports/daily", h.Daily),
		core_http_server.NewRoute(http.MethodGet, "/reports/period", h.Period),
	}
}

func (h *ReportsHTTPHandler) Daily(w http.ResponseWriter, r *http.Request) {
	daily.NewHandler(h.svc).ServeHTTP(w, r)
}

func (h *ReportsHTTPHandler) Period(w http.ResponseWriter, r *http.Request) {
	period.NewHandler(h.svc).ServeHTTP(w, r)
}