package hoist

import (
	"testing"

	"go.uber.org/zap/zapcore"
)

func TestLogLevelFromString_Debug(t *testing.T) {
	level := logLevelFromString("debug")
	if level.Level() != zapcore.DebugLevel {
		t.Errorf("expected DebugLevel, got %v", level.Level())
	}
}

func TestLogLevelFromString_Info(t *testing.T) {
	level := logLevelFromString("info")
	if level.Level() != zapcore.InfoLevel {
		t.Errorf("expected InfoLevel, got %v", level.Level())
	}
}

func TestLogLevelFromString_Warn(t *testing.T) {
	level := logLevelFromString("warn")
	if level.Level() != zapcore.WarnLevel {
		t.Errorf("expected WarnLevel, got %v", level.Level())
	}
}

func TestLogLevelFromString_Error(t *testing.T) {
	level := logLevelFromString("error")
	if level.Level() != zapcore.ErrorLevel {
		t.Errorf("expected ErrorLevel, got %v", level.Level())
	}
}

func TestLogLevelFromString_Invalid(t *testing.T) {
	level := logLevelFromString("invalid")
	if level.Level() != zapcore.InfoLevel {
		t.Errorf("expected InfoLevel for invalid input, got %v", level.Level())
	}
}

func TestSetupLogger_JSONFormat(t *testing.T) {
	cfg := Config{
		LogLevel:  "info",
		LogFormat: "json",
	}

	logger, err := setupLogger(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if logger == nil {
		t.Fatal("expected logger to be non-nil")
	}

	logger.Sync()
}

func TestSetupLogger_TextFormat(t *testing.T) {
	cfg := Config{
		LogLevel:  "debug",
		LogFormat: "text",
	}

	logger, err := setupLogger(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if logger == nil {
		t.Fatal("expected logger to be non-nil")
	}

	logger.Sync()
}

func TestSetupLogger_InvalidFormat(t *testing.T) {
	cfg := Config{
		LogLevel:  "info",
		LogFormat: "xml",
	}

	logger, err := setupLogger(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if logger == nil {
		t.Fatal("expected logger to be non-nil")
	}

	logger.Sync()
}

func TestInitLogger(t *testing.T) {
	cfg := Config{
		LogLevel:  "info",
		LogFormat: "json",
	}

	initLogger(cfg)

	if logger == nil {
		t.Fatal("expected package-level logger to be initialized")
	}

	logger.Sync()
}

func TestGetLogger(t *testing.T) {
	cfg := Config{
		LogLevel:  "info",
		LogFormat: "json",
	}

	initLogger(cfg)

	retrieved := GetLogger()
	if retrieved == nil {
		t.Fatal("expected GetLogger to return non-nil logger")
	}

	if retrieved != logger {
		t.Error("expected GetLogger to return the package-level logger")
	}

	retrieved.Sync()
}

func TestGetLogger_BeforeInit(t *testing.T) {
	logger = nil

	retrieved := GetLogger()
	if retrieved == nil {
		t.Fatal("expected GetLogger to return a default logger even before init")
	}

	retrieved.Sync()
}

func TestSetupLogger_AllLevels(t *testing.T) {
	levels := []string{"debug", "info", "warn", "error"}

	for _, level := range levels {
		cfg := Config{
			LogLevel:  level,
			LogFormat: "json",
		}

		logger, err := setupLogger(cfg)
		if err != nil {
			t.Errorf("unexpected error for level %s: %v", level, err)
		}

		if logger == nil {
			t.Errorf("expected logger to be non-nil for level %s", level)
		}

		logger.Sync()
	}
}

func TestSetupLogger_LogsAtCorrectLevel(t *testing.T) {
	cfg := Config{
		LogLevel:  "warn",
		LogFormat: "json",
	}

	logger, err := setupLogger(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if logger == nil {
		t.Fatal("expected logger to be non-nil")
	}

	if !logger.Core().Enabled(zapcore.WarnLevel) {
		t.Error("expected logger to be enabled at WarnLevel")
	}

	if logger.Core().Enabled(zapcore.InfoLevel) {
		t.Error("expected logger to be disabled at InfoLevel")
	}

	logger.Sync()
}
