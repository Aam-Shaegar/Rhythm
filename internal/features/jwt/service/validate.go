package service

import (
	"time"

	core_errors "github.com/Aam-Shaegar/Rhythm/internal/core/errors"
	"github.com/golang-jwt/jwt/v5"
)

func (s *JwtServiceImpl) ValidateAccessToken(tokenString string) (userID, username string, err error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, core_errors.ErrUnauthorized
		}
		return []byte(s.cfg.JwtAccessSecret), nil
	})

	if err != nil {
		return "", "", core_errors.ErrUnauthorized
	}

	if !token.Valid {
		return "", "", core_errors.ErrUnauthorized
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return "", "", core_errors.ErrUnauthorized
	}

	// Check token type
	tokenType, _ := claims["type"].(string)
	if tokenType != "access" {
		return "", "", core_errors.ErrUnauthorized
	}

	userIDStr, _ := claims["uid"].(string)
	if userIDStr == "" {
		return "", "", core_errors.ErrUnauthorized
	}

	username = claims["uname"].(string)

	// Check expiration
	exp, _ := claims["exp"].(float64)
	if time.Now().UTC().Unix() > int64(exp) {
		return "", "", core_errors.ErrUnauthorized
	}

	return userIDStr, username, nil
}