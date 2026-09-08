package core_config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	TimeZone            *time.Location
	JwtAccessSecret     string
	JwtAccessTTL        time.Duration
	JwtRefreshSecret    string
	JwtRefreshTTL       time.Duration
	RedisAddr           string
	AllowedOrigins      []string
	SecureRefreshCookie bool
}

func NewConfig() (*Config, error) {
	tz := os.Getenv("TIME_ZONE")
	if tz == "" {
		tz = "UTC"
	}
	zone, err := time.LoadLocation(tz)
	if err != nil {
		return nil, fmt.Errorf("load time zone: %s: %w", tz, err)
	}

	accessSecret := os.Getenv("JWT_ACCESS_SECRET")
	if accessSecret == "" {
		return nil, fmt.Errorf("JWT_ACCESS_SECRET is missing")
	}
	accessTTL, err := time.ParseDuration(os.Getenv("JWT_ACCESS_TTL"))
	if err != nil {
		return nil, fmt.Errorf("parse access ttl: %w", err)
	}

	refreshSecret := os.Getenv("JWT_REFRESH_SECRET")
	if refreshSecret == "" {
		return nil, fmt.Errorf("JWT_REFRESH_SECRET is missing")
	}
	refreshTTL, err := time.ParseDuration(os.Getenv("JWT_REFRESH_TTL"))
	if err != nil {
		return nil, fmt.Errorf("parse refresh ttl: %w", err)
	}

	redisAddr := os.Getenv("REDIS_ADDR")
	if redisAddr == "" {
		redisAddr = "localhost:6379"
	}

	allowedOrigins := splitCSV(os.Getenv("CORS_ALLOWED_ORIGINS"))
	if len(allowedOrigins) == 0 {
		allowedOrigins = []string{"http://localhost:5173"}
	}

	secureRefreshCookie := false
	if raw := os.Getenv("SECURE_REFRESH_COOKIE"); raw != "" {
		secureRefreshCookie, err = strconv.ParseBool(raw)
		if err != nil {
			return nil, fmt.Errorf("parse secure refresh cookie: %w", err)
		}
	}

	return &Config{
		TimeZone:            zone,
		JwtAccessSecret:     accessSecret,
		JwtAccessTTL:        accessTTL,
		JwtRefreshSecret:    refreshSecret,
		JwtRefreshTTL:       refreshTTL,
		RedisAddr:           redisAddr,
		AllowedOrigins:      allowedOrigins,
		SecureRefreshCookie: secureRefreshCookie,
	}, nil
}

func NewConfigMust() *Config {
	config, err := NewConfig()
	if err != nil {
		panic(fmt.Errorf("get core config: %w", err))
	}
	return config
}

func splitCSV(raw string) []string {
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	values := make([]string, 0, len(parts))
	for _, part := range parts {
		value := strings.TrimSpace(part)
		if value != "" {
			values = append(values, value)
		}
	}
	return values
}
