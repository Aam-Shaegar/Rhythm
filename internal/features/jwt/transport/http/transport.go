package http

import (
	"net/http"
	"time"

	core_http_server "github.com/Aam-Shaegar/Rhythm/internal/core/transport/http/server"
	"github.com/Aam-Shaegar/Rhythm/internal/features/jwt/service"
)

type JwtHTTPHandler struct {
	svc          service.JwtService
	refreshTTL   time.Duration
	secureCookie bool
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
		core_http_server.NewRoute(http.MethodPost, "/auth/logout", h.Logout),
	}
}

func (h *JwtHTTPHandler) Refresh(w http.ResponseWriter, r *http.Request) {
	NewRefreshHandler(h.svc, h.refreshTTL, h.secureCookie).ServeHTTP(w, r)
}

func (h *JwtHTTPHandler) Logout(w http.ResponseWriter, r *http.Request) {
	if cookie, err := r.Cookie("refresh_token"); err == nil {
		if err := h.svc.RevokeRefreshToken(r.Context(), cookie.Value); err != nil {
			// Clear the browser cookie even when the token was already revoked.
		}
	}
	http.SetCookie(w, &http.Cookie{Name: "refresh_token", Value: "", Path: "/", MaxAge: -1, HttpOnly: true, Secure: h.secureCookie, SameSite: http.SameSiteLaxMode})
	w.WriteHeader(http.StatusNoContent)
}
