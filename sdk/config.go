package main

type Config struct {
	AppName string `json:"app_name"`
	Port int   `json:"port"`
	LogLevel string `json:"log_level"`
	LogFormat string `json:"log_format"`
	MetricsEnabled bool `json:"metrics_enabled"`
	HealthPath string `json:"health_path"`
}

func defaults() Config {
	// Returns a default config 
	return Config{} 
}

func loadConfig() (Config, error) {
	// Main entry point. Calls the other fucntions
	return Config{}, nil
}

func loadJSONFile(path string) (Config, error) {
	// Reads and parses a JSON file. Returns empty config if file doesn't exist (not an error).

	return Config{}, nil
}

func mergeConfigs(base, overide Config) Config {
	// Merges two configs, with the second one overriding the first one.
	return Config{}
}

func validateConfig(cfg Config) error {
	// Validates the config. Returns an error if invalid.
	return nil
}