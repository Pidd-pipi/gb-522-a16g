package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"fiber-otdr-fault-localization/backend/internal/config"
	"fiber-otdr-fault-localization/backend/internal/handler"
	appmw "fiber-otdr-fault-localization/backend/internal/middleware"
	"fiber-otdr-fault-localization/backend/internal/repository"
	"fiber-otdr-fault-localization/backend/internal/router"
	"fiber-otdr-fault-localization/backend/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	cfg, err := config.Load()
	if err != nil {
		logger.Error("configuration_invalid", "error", err)
		os.Exit(1)
	}
	db, err := repository.Open(cfg)
	if err != nil {
		logger.Error("database_open_failed", "error", err)
		os.Exit(1)
	}
	if cfg.DBAutoMigrate {
		if err := repository.MigrateAndSeed(db); err != nil {
			logger.Error("database_prepare_failed", "error", err)
			os.Exit(1)
		}
	}
	store := repository.NewStore(db)
	authService := service.NewAuthService(store, cfg)
	routeService := service.NewRouteService(store)
	traceService := service.NewTraceService(store, cfg.MaxTracePoints)
	eventService := service.NewEventService(store)
	caseService := service.NewCaseService(store)
	auditService := service.NewAuditService(store)
	validate := validator.New(validator.WithRequiredStructEnabled())
	deps := router.Dependencies{
		AuthHandler: handler.NewAuthHandler(authService, validate), RouteHandler: handler.NewFiberRouteHandler(routeService, validate),
		TraceHandler: handler.NewTraceHandler(traceService, validate), EventHandler: handler.NewEventHandler(eventService, validate),
		CaseHandler: handler.NewCaseHandler(caseService, validate), AuditHandler: handler.NewAuditHandler(auditService), AuthService: authService,
		LoginLimiter: appmw.NewRateLimiter(cfg.LoginRateLimit, time.Minute), ImportLimiter: appmw.NewRateLimiter(cfg.ImportRateLimit, time.Minute), AnalyzeLimiter: appmw.NewRateLimiter(cfg.AnalyzeRateLimit, time.Minute),
	}
	gin.SetMode(gin.ReleaseMode)
	engine := gin.New()
	engine.Use(appmw.RequestIDMiddleware(), appmw.AccessLogMiddleware(logger), appmw.RecoveryMiddleware(logger), appmw.CORSMiddleware(cfg.CORSOrigins))
	router.Register(engine, deps)
	server := &http.Server{Addr: ":" + cfg.Port, Handler: engine, ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 20 * time.Second, WriteTimeout: 30 * time.Second, IdleTimeout: 60 * time.Second}
	go func() {
		logger.Info("server_started", "port", cfg.Port, "db_driver", cfg.DBDriver)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("server_failed", "error", err)
			os.Exit(1)
		}
	}()
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop
	ctx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		logger.Error("server_shutdown_failed", "error", err)
	}
	if sqlDB, err := db.DB(); err == nil {
		_ = sqlDB.Close()
	}
	logger.Info("server_stopped")
}
