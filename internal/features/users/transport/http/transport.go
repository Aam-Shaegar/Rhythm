package http

import (
	"net/http"

	core_config "github.com/Aam-Shaegar/Rhythm/internal/core/config"
	core_http_server "github.com/Aam-Shaegar/Rhythm/internal/core/transport/http/server"
	"github.com/Aam-Shaegar/Rhythm/internal/features/users/service"
	"github.com/Aam-Shaegar/Rhythm/internal/features/users/transport/http/get_user"
	"github.com/Aam-Shaegar/Rhythm/internal/features/users/transport/http/login"
	"github.com/Aam-Shaegar/Rhythm/internal/features/users/transport/http/register"
	"github.com/Aam-Shaegar/Rhythm/internal/features/users/transport/http/update_profile"
)

type UsersHTTPHandler struct {
	svc service.UsersService
	cfg *core_config.Config
}

func NewUsersHTTPHandler(svc service.UsersService, cfg *core_config.Config) *UsersHTTPHandler {
	return &UsersHTTPHandler{
		svc: svc,
		cfg: cfg,
	}
}

func (h *UsersHTTPHandler) Routes() []core_http_server.Route {
	return []core_http_server.Route{
		core_http_server.NewRoute(http.MethodPost, "/auth/register", h.Register),
		core_http_server.NewRoute(http.MethodPost, "/auth/login", h.Login),
		core_http_server.NewRoute(http.MethodGet, "/users/me", h.GetMe),
		core_http_server.NewRoute(http.MethodPatch, "/users/me", h.UpdateProfile),
	}
}

func (h *UsersHTTPHandler) Register(w http.ResponseWriter, r *http.Request) {
	register.NewHandler(h.svc).ServeHTTP(w, r)
}

func (h *UsersHTTPHandler) Login(w http.ResponseWriter, r *http.Request) {
	login.NewHandler(h.svc, h.cfg.JwtRefreshTTL, h.cfg.SecureRefreshCookie).ServeHTTP(w, r)
}

func (h *UsersHTTPHandler) GetMe(w http.ResponseWriter, r *http.Request) {
	get_user.NewHandler(h.svc).ServeHTTP(w, r)
}

func (h *UsersHTTPHandler) UpdateProfile(w http.ResponseWriter, r *http.Request) {
	update_profile.NewHandler(h.svc).ServeHTTP(w, r)
}
