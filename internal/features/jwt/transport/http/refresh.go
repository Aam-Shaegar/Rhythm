package http

import (
	"net/http"
	"time"

	core_errors "github.com/Aam-Shaegar/Rhythm/internal/core/errors"
	core_response "github.com/Aam-Shaegar/Rhythm/internal/core/transport/http/response"
	"github.com/Aam-Shaegar/Rhythm/internal/features/jwt/service"
)

type RefreshHandler struct {
	svc          service.JwtService
	refreshTTL   time.Duration
	secureCookie bool
}

func NewRefreshHandler(svc service.JwtService, refreshTTL time.Duration, secureCookie bool) *RefreshHandler {
	return &RefreshHandler{
		svc:          svc,
		refreshTTL:   refreshTTL,
		secureCookie: secureCookie,
	}
}

func (h *RefreshHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	refreshToken := r.Header.Get("X-Refresh-Token")
	if refreshToken == "" {
		cookie, err := r.Cookie("refresh_token")
		if err == nil {
			refreshToken = cookie.Value
		}
	}

	if refreshToken == "" {
		core_response.NewHTTPResponseHandler(nil, w).ErrorResponse(core_errors.ErrUnauthorized, "refresh token required")
		return
	}

	tokenPair, err := h.svc.RefreshTokens(ctx, refreshToken)
	if err != nil {
		core_response.NewHTTPResponseHandler(nil, w).ErrorResponse(err, "failed to refresh tokens")
		return
	}

	h.setRefreshCookie(w, tokenPair.RefreshToken)

	respHandler := core_response.NewHTTPResponseHandler(nil, w)
	respHandler.JSONResponse(map[string]string{
		"access_token": tokenPair.AccessToken,
	}, http.StatusOK)
}

func (h *RefreshHandler) setRefreshCookie(w http.ResponseWriter, token string) {
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
