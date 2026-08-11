package main

import (
	"encoding/json"
	"os"
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
	// Main entry point. Calls the other functions
	return Config{}, nil
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
	// Merges two configs, with the second one overriding the first one.
	return Config{}
}

func validateConfig(cfg Config) error {
	// Validates the config. Returns an error if invalid.
	return nil
}