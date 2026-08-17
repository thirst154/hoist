// Package config loads and validates hoistd daemon configuration.
package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
)

// searchPaths are checked in order; the first hoistd.json found wins.
// It is a var so tests can substitute hermetic paths.
var searchPaths = []string{"hoistd.json", "/etc/hoist/hoistd.json"}

type Config struct {
	SSHAddr          string  `json:"ssh_addr"`
	HTTPAddr         string  `json:"http_addr"`
	DockerHost       string  `json:"docker_host"`
	CaddyAdminURL    string  `json:"caddy_admin_url"`
	BaseDomain       string  `json:"base_domain"`
	DataDir          string  `json:"data_dir"`
	DefaultCPULimit  float64 `json:"default_cpu_limit"`
	DefaultMemoryMB  int     `json:"default_memory_mb"`
	BuildTimeoutSec  int     `json:"build_timeout_sec"`
	HealthTimeoutSec int     `json:"health_timeout_sec"`
	FailBuildOnVulns bool    `json:"fail_build_on_vulns"`
	RetainedImages   int     `json:"retained_images"`
}

// jsonConfig mirrors Config with pointer fields so that JSON file overrides
// can distinguish "field absent" from "field explicitly set to zero value"
// (e.g. "fail_build_on_vulns": false).
type jsonConfig struct {
	SSHAddr          *string  `json:"ssh_addr"`
	HTTPAddr         *string  `json:"http_addr"`
	DockerHost       *string  `json:"docker_host"`
	CaddyAdminURL    *string  `json:"caddy_admin_url"`
	BaseDomain       *string  `json:"base_domain"`
	DataDir          *string  `json:"data_dir"`
	DefaultCPULimit  *float64 `json:"default_cpu_limit"`
	DefaultMemoryMB  *int     `json:"default_memory_mb"`
	BuildTimeoutSec  *int     `json:"build_timeout_sec"`
	HealthTimeoutSec *int     `json:"health_timeout_sec"`
	FailBuildOnVulns *bool    `json:"fail_build_on_vulns"`
	RetainedImages   *int     `json:"retained_images"`
}

func defaults() Config {
	return Config{
		SSHAddr:          ":2222",
		HTTPAddr:         "127.0.0.1:7575",
		DockerHost:       "unix:///var/run/docker.sock",
		CaddyAdminURL:    "http://localhost:2019",
		BaseDomain:       "hoist.local",
		DataDir:          "/var/lib/hoist",
		DefaultCPULimit:  0.5,
		DefaultMemoryMB:  256,
		BuildTimeoutSec:  600,
		HealthTimeoutSec: 30,
		FailBuildOnVulns: false,
		RetainedImages:   3,
	}
}

// Load returns the daemon configuration: built-in defaults overlaid with the
// first hoistd.json found in the search paths. If no file exists, defaults
// are returned.
func Load() (Config, error) {
	for _, path := range searchPaths {
		_, err := os.Stat(path)
		if err == nil {
			return LoadFile(path)
		}
		if !errors.Is(err, os.ErrNotExist) {
			return Config{}, fmt.Errorf("failed to stat %s: %w", path, err)
		}
	}

	cfg := defaults()
	if err := validateConfig(cfg); err != nil {
		return Config{}, fmt.Errorf("invalid default config: %w", err)
	}
	return cfg, nil
}

// LoadFile returns defaults overlaid with the config file at path.
// Unlike Load, a missing or unreadable file is an error.
func LoadFile(path string) (Config, error) {
	cfg := defaults()

	fileCfg, err := loadJSONFile(path)
	if err != nil {
		return Config{}, fmt.Errorf("failed to load %s: %w", path, err)
	}
	cfg = mergeConfigs(cfg, fileCfg)

	if err := validateConfig(cfg); err != nil {
		return Config{}, fmt.Errorf("invalid config in %s: %w", path, err)
	}
	return cfg, nil
}

func loadJSONFile(path string) (jsonConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return jsonConfig{}, err
	}

	var cfg jsonConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return jsonConfig{}, err
	}

	return cfg, nil
}

func mergeConfigs(base Config, override jsonConfig) Config {
	result := base

	if override.SSHAddr != nil {
		result.SSHAddr = *override.SSHAddr
	}
	if override.HTTPAddr != nil {
		result.HTTPAddr = *override.HTTPAddr
	}
	if override.DockerHost != nil {
		result.DockerHost = *override.DockerHost
	}
	if override.CaddyAdminURL != nil {
		result.CaddyAdminURL = *override.CaddyAdminURL
	}
	if override.BaseDomain != nil {
		result.BaseDomain = *override.BaseDomain
	}
	if override.DataDir != nil {
		result.DataDir = *override.DataDir
	}
	if override.DefaultCPULimit != nil {
		result.DefaultCPULimit = *override.DefaultCPULimit
	}
	if override.DefaultMemoryMB != nil {
		result.DefaultMemoryMB = *override.DefaultMemoryMB
	}
	if override.BuildTimeoutSec != nil {
		result.BuildTimeoutSec = *override.BuildTimeoutSec
	}
	if override.HealthTimeoutSec != nil {
		result.HealthTimeoutSec = *override.HealthTimeoutSec
	}
	if override.FailBuildOnVulns != nil {
		result.FailBuildOnVulns = *override.FailBuildOnVulns
	}
	if override.RetainedImages != nil {
		result.RetainedImages = *override.RetainedImages
	}

	return result
}

// domainRegex matches valid lowercase DNS domain names, including single
// labels (e.g. "localhost"). App names become subdomains of base_domain, so
// it must be DNS-safe.
var domainRegex = regexp.MustCompile(`^[a-z0-9]([a-z0-9-]*[a-z0-9])?(\.[a-z0-9]([a-z0-9-]*[a-z0-9])?)*$`)

func validateConfig(cfg Config) error {
	if err := validateAddr("ssh_addr", cfg.SSHAddr); err != nil {
		return err
	}
	if err := validateAddr("http_addr", cfg.HTTPAddr); err != nil {
		return err
	}
	if cfg.DockerHost == "" {
		return fmt.Errorf("docker_host cannot be empty")
	}
	if err := validateCaddyURL(cfg.CaddyAdminURL); err != nil {
		return err
	}
	if !domainRegex.MatchString(cfg.BaseDomain) {
		return fmt.Errorf("base_domain must be a valid domain name, got '%s'", cfg.BaseDomain)
	}
	if !filepath.IsAbs(cfg.DataDir) {
		return fmt.Errorf("data_dir must be an absolute path, got '%s'", cfg.DataDir)
	}
	if cfg.DefaultCPULimit <= 0 {
		return fmt.Errorf("default_cpu_limit must be greater than 0, got %v", cfg.DefaultCPULimit)
	}
	// Docker rejects memory limits below 6MB.
	if cfg.DefaultMemoryMB < 6 {
		return fmt.Errorf("default_memory_mb must be at least 6 (Docker minimum), got %d", cfg.DefaultMemoryMB)
	}
	if cfg.BuildTimeoutSec <= 0 {
		return fmt.Errorf("build_timeout_sec must be greater than 0, got %d", cfg.BuildTimeoutSec)
	}
	if cfg.HealthTimeoutSec <= 0 {
		return fmt.Errorf("health_timeout_sec must be greater than 0, got %d", cfg.HealthTimeoutSec)
	}
	if cfg.RetainedImages < 1 {
		return fmt.Errorf("retained_images must be at least 1, got %d", cfg.RetainedImages)
	}
	return nil
}

func validateAddr(name, addr string) error {
	if addr == "" {
		return fmt.Errorf("%s cannot be empty", name)
	}
	_, port, err := net.SplitHostPort(addr)
	if err != nil {
		return fmt.Errorf("%s must be in host:port form, got '%s'", name, addr)
	}
	p, err := strconv.Atoi(port)
	if err != nil || p < 1 || p > 65535 {
		return fmt.Errorf("%s port must be between 1 and 65535, got '%s'", name, addr)
	}
	return nil
}

func validateCaddyURL(raw string) error {
	if raw == "" {
		return fmt.Errorf("caddy_admin_url cannot be empty")
	}
	u, err := url.Parse(raw)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return fmt.Errorf("caddy_admin_url must be a valid http(s) URL, got '%s'", raw)
	}
	return nil
}
