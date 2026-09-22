package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	core_config "github.com/Aam-Shaegar/Rhythm/internal/core/config"
	core_logger "github.com/Aam-Shaegar/Rhythm/internal/core/logger"
	core_pgx_pool "github.com/Aam-Shaegar/Rhythm/internal/core/repository/postgres/pool/pgx"
	core_http_middleware "github.com/Aam-Shaegar/Rhythm/internal/core/transport/http/middleware"
	core_http_server "github.com/Aam-Shaegar/Rhythm/internal/core/transport/http/server"

	jwt_repository_postgres "github.com/Aam-Shaegar/Rhythm/internal/features/jwt/repository/postgres"
	jwt_service "github.com/Aam-Shaegar/Rhythm/internal/features/jwt/service"
	jwt_transport_http "github.com/Aam-Shaegar/Rhythm/internal/features/jwt/transport/http"

	users_repository_postgres "github.com/Aam-Shaegar/Rhythm/internal/features/users/repository/postgres"
	users_service "github.com/Aam-Shaegar/Rhythm/internal/features/users/service"
	users_transport_http "github.com/Aam-Shaegar/Rhythm/internal/features/users/transport/http"

	events_repository_postgres "github.com/Aam-Shaegar/Rhythm/internal/features/events/repository/postgres"
	events_service "github.com/Aam-Shaegar/Rhythm/internal/features/events/service"
	events_transport_http "github.com/Aam-Shaegar/Rhythm/internal/features/events/transport/http"

	tasks_repository_postgres "github.com/Aam-Shaegar/Rhythm/internal/features/tasks/repository/postgres"
	tasks_service "github.com/Aam-Shaegar/Rhythm/internal/features/tasks/service"
	tasks_transport_http "github.com/Aam-Shaegar/Rhythm/internal/features/tasks/transport/http"

	reports_service "github.com/Aam-Shaegar/Rhythm/internal/features/reports/service"
	reports_transport_http "github.com/Aam-Shaegar/Rhythm/internal/features/reports/transport/http"

	reminders_repository_postgres "github.com/Aam-Shaegar/Rhythm/internal/features/reminders/repository/postgres"
	reminders_service "github.com/Aam-Shaegar/Rhythm/internal/features/reminders/service"
	reminders_transport_http "github.com/Aam-Shaegar/Rhythm/internal/features/reminders/transport/http"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

func main() {
	cfg := core_config.NewConfigMust()
	time.Local = cfg.TimeZone

	ctx, cancel := signal.NotifyContext(
		context.Background(),
		syscall.SIGINT, syscall.SIGTERM,
	)
	defer cancel()

	fmt.Println("Rhytm app starting")

	logger, err := core_logger.NewLogger(core_logger.NewConfigMust())
	if err != nil {
		fmt.Println("failed to init application logger:", err)
		os.Exit(1)
	}
	defer logger.Close()

	logger.Debug("application time zone", zap.Any("zone", time.Local))

	logger.Debug("initializing postgres connection pool")
	pool, err := core_pgx_pool.NewConnectionPool(core_pgx_pool.NewConfigMust(), ctx)
	if err != nil {
		logger.Fatal("failed to init postgres connection pool", zap.Error(err))
	}
	defer pool.Close()

	logger.Debug("initializing redis client")
	redisClient := redis.NewClient(&redis.Options{
		Addr: cfg.RedisAddr,
	})
	if err := redisClient.Ping(ctx).Err(); err != nil {
		logger.Fatal("failed to ping redis", zap.Error(err))
	}
	defer redisClient.Close()

	jwtRepo := jwt_repository_postgres.NewJwtRepository(pool)
	usersRepo := users_repository_postgres.NewUsersRepository(pool)
	eventsRepo := events_repository_postgres.NewEventsRepository(pool)
	tasksRepo := tasks_repository_postgres.NewTasksRepository(pool)
	remindersRepo := reminders_repository_postgres.NewRemindersRepository(pool)

	jwtSvc := jwt_service.NewJwtService(jwtRepo, usersRepo, cfg)
	usersSvc := users_service.NewUsersService(usersRepo, jwtSvc)
	var pushSender reminders_service.PushSender
	if cfg.VapidPublicKey != "" && cfg.VapidPrivateKey != "" {
		pushSender = reminders_service.NewWebPushSender(cfg.VapidPublicKey, cfg.VapidPrivateKey, cfg.VapidSubject)
	} else {
		logger.Warn("VAPID keys missing: push notifications disabled (scheduling still works)")
	}
	remindersSvc := reminders_service.NewRemindersService(remindersRepo, pushSender)
	remindersSvc.SetLogger(logger)
	eventsSvc := events_service.NewEventsService(eventsRepo, remindersSvc)
	tasksSvc := tasks_service.NewTasksService(tasksRepo, remindersSvc)
	reportsSvc := reports_service.NewReportsService(tasksRepo, eventsRepo)

	jwtHandler := jwt_transport_http.NewJwtHTTPHandler(jwtSvc, cfg.JwtRefreshTTL, cfg.SecureRefreshCookie)
	usersHandler := users_transport_http.NewUsersHTTPHandler(usersSvc, cfg)
	eventsHandler := events_transport_http.NewEventsHTTPHandler(eventsSvc)
	tasksHandler := tasks_transport_http.NewTasksHTTPHandler(tasksSvc)
	reportsHandler := reports_transport_http.NewReportsHTTPHandler(reportsSvc)
	remindersHandler := reminders_transport_http.NewRemindersHTTPHandler(remindersSvc)

	authMiddleware := core_http_middleware.Auth(jwtSvc.ValidateAccessToken)

	apiRouter := core_http_server.NewAPIVersionRouter(core_http_server.ApiVersion1)

	publicRoutes := jwtHandler.Routes()
	publicRoutes = append(publicRoutes, usersHandler.Routes()...)
	filteredPublicRoutes := make([]core_http_server.Route, 0)
	for _, route := range publicRoutes {
		if route.Path == "/auth/register" || route.Path == "/auth/login" || route.Path == "/auth/refresh" {
			filteredPublicRoutes = append(filteredPublicRoutes, route)
		}
	}
	apiRouter.RegisterRoutes(filteredPublicRoutes...)

	protectedRoutes := []core_http_server.Route{}
	for _, route := range usersHandler.Routes() {
		if route.Path != "/auth/register" && route.Path != "/auth/login" {
			route.Middleware = append(route.Middleware, authMiddleware)
			protectedRoutes = append(protectedRoutes, route)
		}
	}
	for _, route := range eventsHandler.Routes() {
		route.Middleware = append(route.Middleware, authMiddleware)
		protectedRoutes = append(protectedRoutes, route)
	}
	for _, route := range tasksHandler.Routes() {
		route.Middleware = append(route.Middleware, authMiddleware)
		protectedRoutes = append(protectedRoutes, route)
	}
	for _, route := range reportsHandler.Routes() {
		route.Middleware = append(route.Middleware, authMiddleware)
		protectedRoutes = append(protectedRoutes, route)
	}
	for _, route := range remindersHandler.Routes() {
		route.Middleware = append(route.Middleware, authMiddleware)
		protectedRoutes = append(protectedRoutes, route)
	}
	apiRouter.RegisterRoutes(protectedRoutes...)

	logger.Debug("initializing http server")
	httpServer := core_http_server.NewHTTPServer(
		core_http_server.NewConfigMust(),
		logger,
		core_http_middleware.CORS(cfg.AllowedOrigins...),
		core_http_middleware.RequestID(),
		core_http_middleware.Logger(logger),
		core_http_middleware.Trace(),
		core_http_middleware.Panic(),
	)
	httpServer.RegisterAPIRouters(apiRouter)
	httpServer.RegisterHealth()

	go jwtSvc.StartCleanup(ctx, time.Hour, logger)
	go tasksSvc.GenerateRecurringTasks(ctx, time.Now().Add(30*24*time.Hour))
	go remindersSvc.StartWorker(ctx, time.Minute, logger)

	if err := httpServer.Run(ctx); err != nil {
		logger.Error("HTTP server run error", zap.Error(err))
	}

	logger.Debug("shutdown complete")
}
