package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/thirst154/hoist/daemon/internal/config"
	"github.com/thirst154/hoist/daemon/internal/state"
)

func main() {
	if err := runMain(); err != nil {
		slog.Error("hoistd exited with error", "error", err)
		os.Exit(1)
	}
}

func runMain() error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	return run(ctx)
}

func run(ctx context.Context) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load daemon config: %w", err)
	}

	return runWithConfig(ctx, cfg)
}

func runWithConfig(ctx context.Context, cfg config.Config) error {
	dbPath := filepath.Join(cfg.DataDir, "hoist.db")
	store, err := state.Open(dbPath)
	if err != nil {
		return fmt.Errorf("open state store: %w", err)
	}

	slog.Info("hoistd started", "database", dbPath)
	<-ctx.Done()
	slog.Info("shutdown requested", "reason", ctx.Err())

	if err := store.Close(); err != nil {
		return fmt.Errorf("close state store: %w", err)
	}
	slog.Info("hoistd stopped")
	return nil
}
