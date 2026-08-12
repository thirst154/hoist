package hoist

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
)

type Config struct {
	AppName        string `json:"app_name"`
	Port           int    `json:"port"`
	LogLevel       string `json:"log_level"`
	LogFormat      string `json:"log_format"`
	MetricsEnabled bool   `json:"metrics_enabled"`
	HealthPath     string `json:"health_path"`
}

func defaults() Config {
	return Config{
		AppName:        "app",
		Port:           8080,
		LogLevel:       "info",
		LogFormat:      "json",
		MetricsEnabled: true,
		HealthPath:     "/healthz",
	}
}

func loadConfig() (Config, error) {
	cfg := defaults()

	cfg = applyEnvVars(cfg)

	hoistJSON, err := loadJSONFile("hoist.json")
	if err != nil {
		return Config{}, fmt.Errorf("failed to load hoist.json: %w", err)
	}
	cfg = mergeConfigs(cfg, hoistJSON)

	localJSON, err := loadJSONFile(".hoist.local.json")
	if err != nil {
		return Config{}, fmt.Errorf("failed to load .hoist.local.json: %w", err)
	}
	cfg = mergeConfigs(cfg, localJSON)

	err = validateConfig(cfg)
	if err != nil {
		return Config{}, fmt.Errorf("invalid config: %w", err)
	}

	return cfg, nil
}

func applyEnvVars(cfg Config) Config {
	if v := os.Getenv("HOIST_APP_NAME"); v != "" {
		cfg.AppName = v
	}

	if v := os.Getenv("HOIST_PORT"); v != "" {
		if port, err := strconv.Atoi(v); err == nil {
			cfg.Port = port
		}
	}

	if v := os.Getenv("HOIST_LOG_LEVEL"); v != "" {
		cfg.LogLevel = v
	}

	if v := os.Getenv("HOIST_LOG_FORMAT"); v != "" {
		cfg.LogFormat = v
	}

	if v := os.Getenv("HOIST_METRICS_ENABLED"); v != "" {
		if v == "false" {
			cfg.MetricsEnabled = false
		} else if v == "true" {
			cfg.MetricsEnabled = true
		}
	}

	if v := os.Getenv("HOIST_HEALTH_PATH"); v != "" {
		cfg.HealthPath = v
	}

	return cfg
}

func loadJSONFile(path string) (Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return Config{}, nil
		}
		return Config{}, err
	}

	var cfg Config
	err = json.Unmarshal(data, &cfg)
	if err != nil {
		return Config{}, err
	}

	return cfg, nil
}

func mergeConfigs(base, override Config) Config {
	result := base

	if override.AppName != "" {
		result.AppName = override.AppName
	}
	if override.Port != 0 {
		result.Port = override.Port
	}
	if override.LogLevel != "" {
		result.LogLevel = override.LogLevel
	}
	if override.LogFormat != "" {
		result.LogFormat = override.LogFormat
	}
	if override.MetricsEnabled {
		result.MetricsEnabled = override.MetricsEnabled
	}
	if override.HealthPath != "" {
		result.HealthPath = override.HealthPath
	}

	return result
}

func validateConfig(cfg Config) error {
	if cfg.AppName == "" {
		return fmt.Errorf("app_name cannot be empty")
	}
	if cfg.Port < 1 || cfg.Port > 65535 {
		return fmt.Errorf("port must be between 1 and 65535, got %d", cfg.Port)
	}
	validLogLevels := map[string]bool{"debug": true, "info": true, "warn": true, "error": true}
	if !validLogLevels[cfg.LogLevel] {
		return fmt.Errorf("log_level must be one of: debug, info, warn, error, got '%s'", cfg.LogLevel)
	}
	validLogFormats := map[string]bool{"json": true, "text": true}
	if !validLogFormats[cfg.LogFormat] {
		return fmt.Errorf("log_format must be one of: json, text, got '%s'", cfg.LogFormat)
	}
	if cfg.HealthPath == "" {
		return fmt.Errorf("health_path cannot be empty")
	}
	if cfg.HealthPath[0] != '/' {
		return fmt.Errorf("health_path must start with '/', got '%s'", cfg.HealthPath)
	}
	return nil
}