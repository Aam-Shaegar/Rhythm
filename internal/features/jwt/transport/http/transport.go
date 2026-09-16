package http

import (
	"net/http"
	"time"

	core_http_server "github.com/Aam-Shaegar/Rhythm/internal/core/transport/http/server"
	"github.com/Aam-Shaegar/Rhythm/internal/features/jwt/service"
)

type JwtHTTPHandler struct {
	svc            service.JwtService
	refreshTTL     time.Duration
	secureCookie   bool
}

func NewJwtHTTPHandler(svc service.JwtService, refreshTTL time.Duration, secureCookie bool) *JwtHTTPHandler {
	return &JwtHTTPHandler{
		svc:          svc,
		refreshTTL:   refreshTTL,
		secureCookie: secureCookie,
	}
}

func (h *JwtHTTPHandler) Routes() []core_http_server.Route {
	return []core_http_server.Route{
		core_http_server.NewRoute(http.MethodPost, "/auth/refresh", h.Refresh),
	}
}

func (h *JwtHTTPHandler) Refresh(w http.ResponseWriter, r *http.Request) {
	NewRefreshHandler(h.svc, h.refreshTTL, h.secureCookie).ServeHTTP(w, r)
}