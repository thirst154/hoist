package main

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/thirst154/hoist/daemon/internal/config"
	"github.com/thirst154/hoist/daemon/internal/state"
)

func TestRunWithConfig_ShutdownClosesStore(t *testing.T) {
	dataDir := t.TempDir()
	dbPath := filepath.Join(dataDir, "hoist.db")
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)

	go func() {
		done <- runWithConfig(ctx, config.Config{DataDir: dataDir})
	}()

	// Wait until the database exists, which means runWithConfig has opened it
	// and is ready to wait for shutdown.
	deadline := time.Now().Add(time.Second)
	for {
		if _, err := os.Stat(dbPath); err == nil {
			break
		} else if !os.IsNotExist(err) {
			cancel()
			t.Fatalf("failed to stat database: %v", err)
		}
		if time.Now().After(deadline) {
			cancel()
			t.Fatal("timed out waiting for database to be created")
		}
		select {
		case err := <-done:
			cancel()
			t.Fatalf("daemon exited before shutdown: %v", err)
		default:
		}
		time.Sleep(5 * time.Millisecond)
	}

	cancel()
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("daemon shutdown failed: %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("daemon did not stop after context cancellation")
	}

	// Reopening the same file verifies that shutdown released the BoltDB lock.
	store, err := state.Open(dbPath)
	if err != nil {
		t.Fatalf("database could not be reopened after shutdown: %v", err)
	}
	if err := store.Close(); err != nil {
		t.Fatalf("failed to close reopened database: %v", err)
	}
}

func TestRunWithConfig_OpenStoreFailure(t *testing.T) {
	dataDir := filepath.Join(t.TempDir(), "not-a-directory")
	if err := os.WriteFile(dataDir, []byte("file"), 0600); err != nil {
		t.Fatalf("failed to create file: %v", err)
	}

	err := runWithConfig(context.Background(), config.Config{DataDir: filepath.Join(dataDir, "nested")})
	if err == nil {
		t.Fatal("expected database open to fail")
	}
	if !strings.Contains(err.Error(), "open state store") {
		t.Errorf("expected contextual state-store error, got %v", err)
	}
}
