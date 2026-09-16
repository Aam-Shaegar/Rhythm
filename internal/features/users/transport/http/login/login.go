package login

import (
	"net/http"
	"time"

	core_request "github.com/Aam-Shaegar/Rhythm/internal/core/transport/http/request"
	core_response "github.com/Aam-Shaegar/Rhythm/internal/core/transport/http/response"
	"github.com/Aam-Shaegar/Rhythm/internal/features/users/domain/dtos"
	"github.com/Aam-Shaegar/Rhythm/internal/features/users/service"
)

type Handler struct {
	svc           service.UsersService
	refreshTTL    time.Duration
	secureCookie  bool
}

func NewHandler(svc service.UsersService, refreshTTL time.Duration, secureCookie bool) *Handler {
	return &Handler{
		svc:          svc,
		refreshTTL:   refreshTTL,
		secureCookie: secureCookie,
	}
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var input dtos.LoginInput
	if err := core_request.DecodeAndValidateRequest(r, &input); err != nil {
		core_response.NewHTTPResponseHandler(nil, w).ErrorResponse(err, "invalid request")
		return
	}

	authResp, err := h.svc.Login(ctx, input)
	if err != nil {
		core_response.NewHTTPResponseHandler(nil, w).ErrorResponse(err, "login failed")
		return
	}

	h.setRefreshCookie(w, authResp.RefreshToken)

	respHandler := core_response.NewHTTPResponseHandler(nil, w)
	respHandler.JSONResponse(map[string]string{
		"access_token": authResp.AccessToken,
		"user_id":      authResp.User.ID.String(),
		"username":     authResp.User.Username,
		"email":        authResp.User.Email,
	}, http.StatusOK)
}

func (h *Handler) setRefreshCookie(w http.ResponseWriter, token string) {
	http.SetCookie(w, &http.Cookie{
		Name:     "refresh_token",
		Value:    token,
		Path:     "/",
		MaxAge:   int(h.refreshTTL.Seconds()),
		HttpOnly: true,
		Secure:   h.secureCookie,
		SameSite: http.SameSiteLaxMode,
	})
}