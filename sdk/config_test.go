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

func TestMergeConfigs_EmptyOverride(t *testing.T) {
	base := Config{
		AppName:        "base-api",
		Port:           8080,
		LogLevel:       "info",
		LogFormat:      "json",
		MetricsEnabled: true,
		HealthPath:     "/healthz",
	}
	override := Config{}

	result := mergeConfigs(base, override)

	if result.AppName != "base-api" {
		t.Errorf("expected AppName 'base-api', got '%s'", result.AppName)
	}
	if result.Port != 8080 {
		t.Errorf("expected Port 8080, got %d", result.Port)
	}
	if result.LogLevel != "info" {
		t.Errorf("expected LogLevel 'info', got '%s'", result.LogLevel)
	}
	if result.LogFormat != "json" {
		t.Errorf("expected LogFormat 'json', got '%s'", result.LogFormat)
	}
	if result.MetricsEnabled != true {
		t.Errorf("expected MetricsEnabled true, got %v", result.MetricsEnabled)
	}
	if result.HealthPath != "/healthz" {
		t.Errorf("expected HealthPath '/healthz', got '%s'", result.HealthPath)
	}
}

func TestMergeConfigs_FullOverride(t *testing.T) {
	base := Config{
		AppName:        "base-api",
		Port:           8080,
		LogLevel:       "info",
		LogFormat:      "json",
		MetricsEnabled: false,
		HealthPath:     "/healthz",
	}
	override := Config{
		AppName:        "override-api",
		Port:           3000,
		LogLevel:       "debug",
		LogFormat:      "text",
		MetricsEnabled: true,
		HealthPath:     "/health",
	}

	result := mergeConfigs(base, override)

	if result.AppName != "override-api" {
		t.Errorf("expected AppName 'override-api', got '%s'", result.AppName)
	}
	if result.Port != 3000 {
		t.Errorf("expected Port 3000, got %d", result.Port)
	}
	if result.LogLevel != "debug" {
		t.Errorf("expected LogLevel 'debug', got '%s'", result.LogLevel)
	}
	if result.LogFormat != "text" {
		t.Errorf("expected LogFormat 'text', got '%s'", result.LogFormat)
	}
	if result.MetricsEnabled != true {
		t.Errorf("expected MetricsEnabled true, got %v", result.MetricsEnabled)
	}
	if result.HealthPath != "/health" {
		t.Errorf("expected HealthPath '/health', got '%s'", result.HealthPath)
	}
}

func TestMergeConfigs_PartialOverride(t *testing.T) {
	base := Config{
		AppName:        "base-api",
		Port:           8080,
		LogLevel:       "info",
		LogFormat:      "json",
		MetricsEnabled: true,
		HealthPath:     "/healthz",
	}
	override := Config{
		Port:     9000,
		LogLevel: "debug",
	}

	result := mergeConfigs(base, override)

	if result.AppName != "base-api" {
		t.Errorf("expected AppName 'base-api', got '%s'", result.AppName)
	}
	if result.Port != 9000 {
		t.Errorf("expected Port 9000, got %d", result.Port)
	}
	if result.LogLevel != "debug" {
		t.Errorf("expected LogLevel 'debug', got '%s'", result.LogLevel)
	}
	if result.LogFormat != "json" {
		t.Errorf("expected LogFormat 'json', got '%s'", result.LogFormat)
	}
	if result.MetricsEnabled != true {
		t.Errorf("expected MetricsEnabled true, got %v", result.MetricsEnabled)
	}
	if result.HealthPath != "/healthz" {
		t.Errorf("expected HealthPath '/healthz', got '%s'", result.HealthPath)
	}
}

func TestMergeConfigs_BothEmpty(t *testing.T) {
	base := Config{}
	override := Config{}

	result := mergeConfigs(base, override)

	if result != (Config{}) {
		t.Errorf("expected empty Config, got %+v", result)
	}
}

func TestMergeConfigs_EmptyBase(t *testing.T) {
	base := Config{}
	override := Config{
		AppName: "override-api",
		Port:    3000,
	}

	result := mergeConfigs(base, override)

	if result.AppName != "override-api" {
		t.Errorf("expected AppName 'override-api', got '%s'", result.AppName)
	}
	if result.Port != 3000 {
		t.Errorf("expected Port 3000, got %d", result.Port)
	}
}

func TestValidateConfig_ValidConfig(t *testing.T) {
	cfg := Config{
		AppName:        "my-api",
		Port:           8080,
		LogLevel:       "info",
		LogFormat:      "json",
		MetricsEnabled: true,
		HealthPath:     "/healthz",
	}

	err := validateConfig(cfg)
	if err != nil {
		t.Errorf("expected no error for valid config, got %v", err)
	}
}

func TestValidateConfig_EmptyAppName(t *testing.T) {
	cfg := Config{
		AppName:        "",
		Port:           8080,
		LogLevel:       "info",
		LogFormat:      "json",
		MetricsEnabled: true,
		HealthPath:     "/healthz",
	}

	err := validateConfig(cfg)
	if err == nil {
		t.Errorf("expected error for empty AppName, got nil")
	}
}

func TestValidateConfig_InvalidPort_Zero(t *testing.T) {
	cfg := Config{
		AppName:        "my-api",
		Port:           0,
		LogLevel:       "info",
		LogFormat:      "json",
		MetricsEnabled: true,
		HealthPath:     "/healthz",
	}

	err := validateConfig(cfg)
	if err == nil {
		t.Errorf("expected error for port 0, got nil")
	}
}

func TestValidateConfig_InvalidPort_Negative(t *testing.T) {
	cfg := Config{
		AppName:        "my-api",
		Port:           -1,
		LogLevel:       "info",
		LogFormat:      "json",
		MetricsEnabled: true,
		HealthPath:     "/healthz",
	}

	err := validateConfig(cfg)
	if err == nil {
		t.Errorf("expected error for negative port, got nil")
	}
}

func TestValidateConfig_InvalidPort_TooHigh(t *testing.T) {
	cfg := Config{
		AppName:        "my-api",
		Port:           70000,
		LogLevel:       "info",
		LogFormat:      "json",
		MetricsEnabled: true,
		HealthPath:     "/healthz",
	}

	err := validateConfig(cfg)
	if err == nil {
		t.Errorf("expected error for port > 65535, got nil")
	}
}

func TestValidateConfig_InvalidLogLevel(t *testing.T) {
	cfg := Config{
		AppName:        "my-api",
		Port:           8080,
		LogLevel:       "verbose",
		LogFormat:      "json",
		MetricsEnabled: true,
		HealthPath:     "/healthz",
	}

	err := validateConfig(cfg)
	if err == nil {
		t.Errorf("expected error for invalid log level, got nil")
	}
}

func TestValidateConfig_ValidLogLevels(t *testing.T) {
	levels := []string{"debug", "info", "warn", "error"}

	for _, level := range levels {
		cfg := Config{
			AppName:        "my-api",
			Port:           8080,
			LogLevel:       level,
			LogFormat:      "json",
			MetricsEnabled: true,
			HealthPath:     "/healthz",
		}

		err := validateConfig(cfg)
		if err != nil {
			t.Errorf("expected no error for log level '%s', got %v", level, err)
		}
	}
}

func TestValidateConfig_InvalidLogFormat(t *testing.T) {
	cfg := Config{
		AppName:        "my-api",
		Port:           8080,
		LogLevel:       "info",
		LogFormat:      "xml",
		MetricsEnabled: true,
		HealthPath:     "/healthz",
	}

	err := validateConfig(cfg)
	if err == nil {
		t.Errorf("expected error for invalid log format, got nil")
	}
}

func TestValidateConfig_ValidLogFormats(t *testing.T) {
	formats := []string{"json", "text"}

	for _, format := range formats {
		cfg := Config{
			AppName:        "my-api",
			Port:           8080,
			LogLevel:       "info",
			LogFormat:      format,
			MetricsEnabled: true,
			HealthPath:     "/healthz",
		}

		err := validateConfig(cfg)
		if err != nil {
			t.Errorf("expected no error for log format '%s', got %v", format, err)
		}
	}
}

func TestValidateConfig_EmptyHealthPath(t *testing.T) {
	cfg := Config{
		AppName:        "my-api",
		Port:           8080,
		LogLevel:       "info",
		LogFormat:      "json",
		MetricsEnabled: true,
		HealthPath:     "",
	}

	err := validateConfig(cfg)
	if err == nil {
		t.Errorf("expected error for empty health path, got nil")
	}
}

func TestValidateConfig_HealthPathNoSlash(t *testing.T) {
	cfg := Config{
		AppName:        "my-api",
		Port:           8080,
		LogLevel:       "info",
		LogFormat:      "json",
		MetricsEnabled: true,
		HealthPath:     "healthz",
	}

	err := validateConfig(cfg)
	if err == nil {
		t.Errorf("expected error for health path without leading slash, got nil")
	}
}

func TestApplyEnvVars_NoEnvVars(t *testing.T) {
	os.Clearenv()

	cfg := Config{
		AppName:        "my-api",
		Port:           8080,
		LogLevel:       "info",
		LogFormat:      "json",
		MetricsEnabled: true,
		HealthPath:     "/healthz",
	}

	result := applyEnvVars(cfg)

	if result.AppName != "my-api" {
		t.Errorf("expected AppName 'my-api', got '%s'", result.AppName)
	}
	if result.Port != 8080 {
		t.Errorf("expected Port 8080, got %d", result.Port)
	}
}

func TestApplyEnvVars_AllEnvVars(t *testing.T) {
	os.Clearenv()
	os.Setenv("HOIST_APP_NAME", "env-api")
	os.Setenv("HOIST_PORT", "3000")
	os.Setenv("HOIST_LOG_LEVEL", "debug")
	os.Setenv("HOIST_LOG_FORMAT", "text")
	os.Setenv("HOIST_METRICS_ENABLED", "false")
	os.Setenv("HOIST_HEALTH_PATH", "/health")

	cfg := defaults()
	result := applyEnvVars(cfg)

	if result.AppName != "env-api" {
		t.Errorf("expected AppName 'env-api', got '%s'", result.AppName)
	}
	if result.Port != 3000 {
		t.Errorf("expected Port 3000, got %d", result.Port)
	}
	if result.LogLevel != "debug" {
		t.Errorf("expected LogLevel 'debug', got '%s'", result.LogLevel)
	}
	if result.LogFormat != "text" {
		t.Errorf("expected LogFormat 'text', got '%s'", result.LogFormat)
	}
	if result.MetricsEnabled != false {
		t.Errorf("expected MetricsEnabled false, got %v", result.MetricsEnabled)
	}
	if result.HealthPath != "/health" {
		t.Errorf("expected HealthPath '/health', got '%s'", result.HealthPath)
	}
}

func TestApplyEnvVars_PartialEnvVars(t *testing.T) {
	os.Clearenv()
	os.Setenv("HOIST_PORT", "9000")
	os.Setenv("HOIST_LOG_LEVEL", "warn")

	cfg := defaults()
	result := applyEnvVars(cfg)

	if result.AppName != "app" {
		t.Errorf("expected AppName 'app', got '%s'", result.AppName)
	}
	if result.Port != 9000 {
		t.Errorf("expected Port 9000, got %d", result.Port)
	}
	if result.LogLevel != "warn" {
		t.Errorf("expected LogLevel 'warn', got '%s'", result.LogLevel)
	}
	if result.LogFormat != "json" {
		t.Errorf("expected LogFormat 'json', got '%s'", result.LogFormat)
	}
}

func TestApplyEnvVars_InvalidPort(t *testing.T) {
	os.Clearenv()
	os.Setenv("HOIST_PORT", "not-a-number")

	cfg := defaults()
	result := applyEnvVars(cfg)

	if result.Port != 8080 {
		t.Errorf("expected Port to remain 8080 for invalid env var, got %d", result.Port)
	}
}

func TestLoadConfig_DefaultsOnly(t *testing.T) {
	os.Clearenv()
	tmpDir := t.TempDir()
	origDir, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(origDir)

	cfg, err := loadConfig()
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	if cfg.AppName != "app" {
		t.Errorf("expected AppName 'app', got '%s'", cfg.AppName)
	}
	if cfg.Port != 8080 {
		t.Errorf("expected Port 8080, got %d", cfg.Port)
	}
}

func TestLoadConfig_WithHoistJSON(t *testing.T) {
	os.Clearenv()
	tmpDir := t.TempDir()
	origDir, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(origDir)

	content := `{
		"app_name": "json-api",
		"port": 3000
	}`
	os.WriteFile("hoist.json", []byte(content), 0644)

	cfg, err := loadConfig()
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	if cfg.AppName != "json-api" {
		t.Errorf("expected AppName 'json-api', got '%s'", cfg.AppName)
	}
	if cfg.Port != 3000 {
		t.Errorf("expected Port 3000, got %d", cfg.Port)
	}
	if cfg.LogLevel != "info" {
		t.Errorf("expected LogLevel 'info' from defaults, got '%s'", cfg.LogLevel)
	}
}

func TestLoadConfig_WithLocalOverride(t *testing.T) {
	os.Clearenv()
	tmpDir := t.TempDir()
	origDir, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(origDir)

	hoistJSON := `{
		"app_name": "json-api",
		"port": 3000,
		"log_level": "info"
	}`
	os.WriteFile("hoist.json", []byte(hoistJSON), 0644)

	localJSON := `{
		"port": 9000,
		"log_level": "debug"
	}`
	os.WriteFile(".hoist.local.json", []byte(localJSON), 0644)

	cfg, err := loadConfig()
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	if cfg.AppName != "json-api" {
		t.Errorf("expected AppName 'json-api', got '%s'", cfg.AppName)
	}
	if cfg.Port != 9000 {
		t.Errorf("expected Port 9000 (from local), got %d", cfg.Port)
	}
	if cfg.LogLevel != "debug" {
		t.Errorf("expected LogLevel 'debug' (from local), got '%s'", cfg.LogLevel)
	}
}

func TestLoadConfig_JSONOverridesEnvVars(t *testing.T) {
	os.Clearenv()
	tmpDir := t.TempDir()
	origDir, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(origDir)

	content := `{
		"app_name": "json-api",
		"port": 3000
	}`
	os.WriteFile("hoist.json", []byte(content), 0644)

	os.Setenv("HOIST_PORT", "4000")

	cfg, err := loadConfig()
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	if cfg.AppName != "json-api" {
		t.Errorf("expected AppName 'json-api', got '%s'", cfg.AppName)
	}
	if cfg.Port != 3000 {
		t.Errorf("expected Port 3000 (from JSON, overriding env), got %d", cfg.Port)
	}
}

func TestLoadConfig_FullPrecedence(t *testing.T) {
	os.Clearenv()
	tmpDir := t.TempDir()
	origDir, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(origDir)

	hoistJSON := `{
		"app_name": "json-api",
		"port": 3000,
		"log_level": "info"
	}`
	os.WriteFile("hoist.json", []byte(hoistJSON), 0644)

	localJSON := `{
		"port": 9000,
		"log_level": "debug"
	}`
	os.WriteFile(".hoist.local.json", []byte(localJSON), 0644)

	os.Setenv("HOIST_LOG_LEVEL", "warn")

	cfg, err := loadConfig()
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	if cfg.AppName != "json-api" {
		t.Errorf("expected AppName 'json-api', got '%s'", cfg.AppName)
	}
	if cfg.Port != 9000 {
		t.Errorf("expected Port 9000 (from local, highest precedence), got %d", cfg.Port)
	}
	if cfg.LogLevel != "debug" {
		t.Errorf("expected LogLevel 'debug' (from local, overriding env), got '%s'", cfg.LogLevel)
	}
}

func TestLoadConfig_InvalidConfig(t *testing.T) {
	os.Clearenv()
	tmpDir := t.TempDir()
	origDir, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(origDir)

	content := `{
		"port": 70000
	}`
	os.WriteFile("hoist.json", []byte(content), 0644)

	_, err := loadConfig()
	if err == nil {
		t.Errorf("expected validation error for invalid port, got nil")
	}
}
