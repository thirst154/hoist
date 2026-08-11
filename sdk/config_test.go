package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaults(t *testing.T) {
	cfg := defaults()

	if cfg.AppName != "app" {
		t.Errorf("expected AppName to be 'app', got '%s'", cfg.AppName)
	}

	if cfg.Port != 8080 {
		t.Errorf("expected Port to be 8080, got %d", cfg.Port)
	}

	if cfg.LogLevel != "info" {
		t.Errorf("expected LogLevel to be 'info', got '%s'", cfg.LogLevel)
	}

	if cfg.LogFormat != "json" {
		t.Errorf("expected LogFormat to be 'json', got '%s'", cfg.LogFormat)
	}

	if cfg.MetricsEnabled != true {
		t.Errorf("expected MetricsEnabled to be true, got %v", cfg.MetricsEnabled)
	}

	if cfg.HealthPath != "/healthz" {
		t.Errorf("expected HealthPath to be '/healthz', got '%s'", cfg.HealthPath)
	}
}

func TestLoadJSONFile_FileDoesNotExist(t *testing.T) {
	cfg, err := loadJSONFile("/nonexistent/path/hoist.json")

	if err != nil {
		t.Errorf("expected no error for missing file, got %v", err)
	}

	if cfg != (Config{}) {
		t.Errorf("expected empty Config for missing file, got %+v", cfg)
	}
}

func TestLoadJSONFile_ValidJSON(t *testing.T) {
	tmpDir := t.TempDir()
	jsonPath := filepath.Join(tmpDir, "hoist.json")

	content := `{
		"app_name": "test-api",
		"port": 3000,
		"log_level": "debug",
		"log_format": "text",
		"metrics_enabled": false,
		"health_path": "/health"
	}`

	err := os.WriteFile(jsonPath, []byte(content), 0644)
	if err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	cfg, err := loadJSONFile(jsonPath)

	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	if cfg.AppName != "test-api" {
		t.Errorf("expected AppName 'test-api', got '%s'", cfg.AppName)
	}

	if cfg.Port != 3000 {
		t.Errorf("expected Port 3000, got %d", cfg.Port)
	}

	if cfg.LogLevel != "debug" {
		t.Errorf("expected LogLevel 'debug', got '%s'", cfg.LogLevel)
	}

	if cfg.LogFormat != "text" {
		t.Errorf("expected LogFormat 'text', got '%s'", cfg.LogFormat)
	}

	if cfg.MetricsEnabled != false {
		t.Errorf("expected MetricsEnabled false, got %v", cfg.MetricsEnabled)
	}

	if cfg.HealthPath != "/health" {
		t.Errorf("expected HealthPath '/health', got '%s'", cfg.HealthPath)
	}
}

func TestLoadJSONFile_PartialJSON(t *testing.T) {
	tmpDir := t.TempDir()
	jsonPath := filepath.Join(tmpDir, "hoist.json")

	content := `{
		"app_name": "partial-api",
		"port": 9000
	}`

	err := os.WriteFile(jsonPath, []byte(content), 0644)
	if err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	cfg, err := loadJSONFile(jsonPath)

	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	if cfg.AppName != "partial-api" {
		t.Errorf("expected AppName 'partial-api', got '%s'", cfg.AppName)
	}

	if cfg.Port != 9000 {
		t.Errorf("expected Port 9000, got %d", cfg.Port)
	}

	if cfg.LogLevel != "" {
		t.Errorf("expected LogLevel to be empty, got '%s'", cfg.LogLevel)
	}

	if cfg.MetricsEnabled != false {
		t.Errorf("expected MetricsEnabled to be false, got %v", cfg.MetricsEnabled)
	}
}

func TestLoadJSONFile_MalformedJSON(t *testing.T) {
	tmpDir := t.TempDir()
	jsonPath := filepath.Join(tmpDir, "hoist.json")

	content := `{
		"app_name": "test-api",
		"port": "not-a-number"
	}`

	err := os.WriteFile(jsonPath, []byte(content), 0644)
	if err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	_, err = loadJSONFile(jsonPath)

	if err == nil {
		t.Errorf("expected error for malformed JSON, got nil")
	}
}

func TestLoadJSONFile_EmptyFile(t *testing.T) {
	tmpDir := t.TempDir()
	jsonPath := filepath.Join(tmpDir, "hoist.json")

	err := os.WriteFile(jsonPath, []byte(""), 0644)
	if err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	_, err = loadJSONFile(jsonPath)

	if err == nil {
		t.Errorf("expected error for empty file, got nil")
	}
}
