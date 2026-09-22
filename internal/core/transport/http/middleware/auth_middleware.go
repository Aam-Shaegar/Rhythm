package core_http_middleware

import (
	"context"
	"net/http"
	"strings"

	core_errors "github.com/Aam-Shaegar/Rhythm/internal/core/errors"
	core_logger "github.com/Aam-Shaegar/Rhythm/internal/core/logger"
	core_response "github.com/Aam-Shaegar/Rhythm/internal/core/transport/http/response"
)

type TokenValidator func(tokenString string) (userID, username string, err error)

func Auth(validator TokenValidator) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			log := core_logger.FromContext(ctx)

			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				core_response.NewHTTPResponseHandler(log, w).ErrorResponse(
					core_errors.ErrUnauthorized, "authorization header required")
				return
			}

			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
				core_response.NewHTTPResponseHandler(log, w).ErrorResponse(
					core_errors.ErrUnauthorized, "invalid authorization header format")
				return
			}

			userID, username, err := validator(parts[1])
			if err != nil {
				core_response.NewHTTPResponseHandler(log, w).ErrorResponse(err, "invalid access token")
				return
			}

			ctx = context.WithValue(ctx, userIDKey, userID)
			ctx = context.WithValue(ctx, usernameKey, username)
			// Старые хендлеры читают строковый ключ "user_id", дублируем значение.
			ctx = context.WithValue(ctx, "user_id", userID)
			ctx = context.WithValue(ctx, "username", username)

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

type contextKey string

const (
	userIDKey   contextKey = "user_id"
	usernameKey contextKey = "username"
)

func GetUserID(ctx context.Context) (string, bool) {
	val := ctx.Value(userIDKey)
	if val == nil {
		return "", false
	}
	id, ok := val.(string)
	return id, ok
}

func GetUsername(ctx context.Context) (string, bool) {
	val := ctx.Value(usernameKey)
	if val == nil {
		return "", false
	}
	username, ok := val.(string)
	return username, ok
}
