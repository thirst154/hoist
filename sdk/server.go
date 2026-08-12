package hoist

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go.uber.org/zap"
)

var (
	currentServer *http.Server
	currentConfig Config
)

func Start(handler http.Handler) {
	cfg, err := loadConfig()
	if err != nil {
		panic(fmt.Sprintf("failed to load config: %v", err))
	}

	currentConfig = cfg

	initLogger(cfg)
	initMetrics(cfg)
	initHealthChecker()

	routes := setupRoutes(handler, cfg)
	wrapped := wrapHandler(routes, cfg)

	server := startServer(wrapped, cfg)
	currentServer = server

	logger.Info("server started",
		zap.String("app_name", cfg.AppName),
		zap.Int("port", cfg.Port),
	)

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGTERM, syscall.SIGINT)

	gracefulShutdown(server, stop)
}

func setupRoutes(handler http.Handler, cfg Config) http.Handler {
	if cfg.HealthPath == "" {
		panic("health_path cannot be empty")
	}

	mux := http.NewServeMux()

	if healthChecker != nil {
		mux.Handle(cfg.HealthPath, healthChecker.Handler())
	}

	if cfg.MetricsEnabled && metrics != nil {
		mux.Handle("/metrics", metrics.Handler())
	}

	mux.Handle("/", handler)

	return mux
}

func startServer(handler http.Handler, cfg Config) *http.Server {
	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Port),
		Handler:      handler,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("server error", zap.Error(err))
		}
	}()

	return server
}

func gracefulShutdown(server *http.Server, stop chan os.Signal) {
	<-stop

	logger.Info("shutdown signal received, starting graceful shutdown")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		logger.Error("shutdown error", zap.Error(err))
	}

	logger.Info("server shutdown complete")
}
