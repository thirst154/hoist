package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func strPtr(s string) *string     { return &s }
func intPtr(i int) *int           { return &i }
func boolPtr(b bool) *bool        { return &b }
func floatPtr(f float64) *float64 { return &f }

func TestDefaults(t *testing.T) {
	cfg := defaults()

	if cfg.SSHAddr != ":2222" {
		t.Errorf("expected SSHAddr ':2222', got '%s'", cfg.SSHAddr)
	}
	if cfg.HTTPAddr != "127.0.0.1:7575" {
		t.Errorf("expected HTTPAddr '127.0.0.1:7575', got '%s'", cfg.HTTPAddr)
	}
	if cfg.DockerHost != "unix:///var/run/docker.sock" {
		t.Errorf("expected DockerHost 'unix:///var/run/docker.sock', got '%s'", cfg.DockerHost)
	}
	if cfg.CaddyAdminURL != "http://localhost:2019" {
		t.Errorf("expected CaddyAdminURL 'http://localhost:2019', got '%s'", cfg.CaddyAdminURL)
	}
	if cfg.BaseDomain != "hoist.local" {
		t.Errorf("expected BaseDomain 'hoist.local', got '%s'", cfg.BaseDomain)
	}
	if cfg.DataDir != "/var/lib/hoist" {
		t.Errorf("expected DataDir '/var/lib/hoist', got '%s'", cfg.DataDir)
	}
	if cfg.DefaultCPULimit != 0.5 {
		t.Errorf("expected DefaultCPULimit 0.5, got %v", cfg.DefaultCPULimit)
	}
	if cfg.DefaultMemoryMB != 256 {
		t.Errorf("expected DefaultMemoryMB 256, got %d", cfg.DefaultMemoryMB)
	}
	if cfg.BuildTimeoutSec != 600 {
		t.Errorf("expected BuildTimeoutSec 600, got %d", cfg.BuildTimeoutSec)
	}
	if cfg.HealthTimeoutSec != 30 {
		t.Errorf("expected HealthTimeoutSec 30, got %d", cfg.HealthTimeoutSec)
	}
	if cfg.FailBuildOnVulns != false {
		t.Errorf("expected FailBuildOnVulns false, got %v", cfg.FailBuildOnVulns)
	}
	if cfg.RetainedImages != 3 {
		t.Errorf("expected RetainedImages 3, got %d", cfg.RetainedImages)
	}
}

func TestLoadJSONFile_FileDoesNotExist(t *testing.T) {
	_, err := loadJSONFile("/nonexistent/path/hoistd.json")

	if err == nil {
		t.Errorf("expected error for missing file, got nil")
	}
}

func TestLoadJSONFile_ValidJSON(t *testing.T) {
	tmpDir := t.TempDir()
	jsonPath := filepath.Join(tmpDir, "hoistd.json")

	content := `{
		"ssh_addr": ":2223",
		"http_addr": "0.0.0.0:7575",
		"docker_host": "tcp://docker:2375",
		"caddy_admin_url": "http://caddy:2019",
		"base_domain": "example.com",
		"data_dir": "/data/hoist",
		"default_cpu_limit": 1.5,
		"default_memory_mb": 512,
		"build_timeout_sec": 300,
		"health_timeout_sec": 60,
		"fail_build_on_vulns": true,
		"retained_images": 5
	}`

	if err := os.WriteFile(jsonPath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	cfg, err := loadJSONFile(jsonPath)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if cfg.SSHAddr == nil || *cfg.SSHAddr != ":2223" {
		t.Errorf("expected SSHAddr ':2223', got %v", cfg.SSHAddr)
	}
	if cfg.HTTPAddr == nil || *cfg.HTTPAddr != "0.0.0.0:7575" {
		t.Errorf("expected HTTPAddr '0.0.0.0:7575', got %v", cfg.HTTPAddr)
	}
	if cfg.DockerHost == nil || *cfg.DockerHost != "tcp://docker:2375" {
		t.Errorf("expected DockerHost 'tcp://docker:2375', got %v", cfg.DockerHost)
	}
	if cfg.CaddyAdminURL == nil || *cfg.CaddyAdminURL != "http://caddy:2019" {
		t.Errorf("expected CaddyAdminURL 'http://caddy:2019', got %v", cfg.CaddyAdminURL)
	}
	if cfg.BaseDomain == nil || *cfg.BaseDomain != "example.com" {
		t.Errorf("expected BaseDomain 'example.com', got %v", cfg.BaseDomain)
	}
	if cfg.DataDir == nil || *cfg.DataDir != "/data/hoist" {
		t.Errorf("expected DataDir '/data/hoist', got %v", cfg.DataDir)
	}
	if cfg.DefaultCPULimit == nil || *cfg.DefaultCPULimit != 1.5 {
		t.Errorf("expected DefaultCPULimit 1.5, got %v", cfg.DefaultCPULimit)
	}
	if cfg.DefaultMemoryMB == nil || *cfg.DefaultMemoryMB != 512 {
		t.Errorf("expected DefaultMemoryMB 512, got %v", cfg.DefaultMemoryMB)
	}
	if cfg.BuildTimeoutSec == nil || *cfg.BuildTimeoutSec != 300 {
		t.Errorf("expected BuildTimeoutSec 300, got %v", cfg.BuildTimeoutSec)
	}
	if cfg.HealthTimeoutSec == nil || *cfg.HealthTimeoutSec != 60 {
		t.Errorf("expected HealthTimeoutSec 60, got %v", cfg.HealthTimeoutSec)
	}
	if cfg.FailBuildOnVulns == nil || *cfg.FailBuildOnVulns != true {
		t.Errorf("expected FailBuildOnVulns true, got %v", cfg.FailBuildOnVulns)
	}
	if cfg.RetainedImages == nil || *cfg.RetainedImages != 5 {
		t.Errorf("expected RetainedImages 5, got %v", cfg.RetainedImages)
	}
}

func TestLoadJSONFile_PartialJSON(t *testing.T) {
	tmpDir := t.TempDir()
	jsonPath := filepath.Join(tmpDir, "hoistd.json")

	content := `{
		"base_domain": "example.com",
		"default_memory_mb": 512
	}`

	if err := os.WriteFile(jsonPath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	cfg, err := loadJSONFile(jsonPath)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if cfg.BaseDomain == nil || *cfg.BaseDomain != "example.com" {
		t.Errorf("expected BaseDomain 'example.com', got %v", cfg.BaseDomain)
	}
	if cfg.DefaultMemoryMB == nil || *cfg.DefaultMemoryMB != 512 {
		t.Errorf("expected DefaultMemoryMB 512, got %v", cfg.DefaultMemoryMB)
	}
	if cfg.SSHAddr != nil {
		t.Errorf("expected SSHAddr to be nil (unset), got %v", *cfg.SSHAddr)
	}
	if cfg.FailBuildOnVulns != nil {
		t.Errorf("expected FailBuildOnVulns to be nil (unset), got %v", *cfg.FailBuildOnVulns)
	}
}

func TestLoadJSONFile_MalformedJSON(t *testing.T) {
	tmpDir := t.TempDir()
	jsonPath := filepath.Join(tmpDir, "hoistd.json")

	content := `{
		"default_memory_mb": "not-a-number"
	}`

	if err := os.WriteFile(jsonPath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	_, err := loadJSONFile(jsonPath)
	if err == nil {
		t.Errorf("expected error for malformed JSON, got nil")
	}
}

func TestLoadJSONFile_EmptyFile(t *testing.T) {
	tmpDir := t.TempDir()
	jsonPath := filepath.Join(tmpDir, "hoistd.json")

	if err := os.WriteFile(jsonPath, []byte(""), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	_, err := loadJSONFile(jsonPath)
	if err == nil {
		t.Errorf("expected error for empty file, got nil")
	}
}

func TestMergeConfigs_EmptyOverride(t *testing.T) {
	base := defaults()
	override := jsonConfig{}

	result := mergeConfigs(base, override)

	if result != base {
		t.Errorf("expected result to equal base, got %+v", result)
	}
}

func TestMergeConfigs_FullOverride(t *testing.T) {
	base := defaults()
	override := jsonConfig{
		SSHAddr:          strPtr(":2223"),
		HTTPAddr:         strPtr("0.0.0.0:7575"),
		DockerHost:       strPtr("tcp://docker:2375"),
		CaddyAdminURL:    strPtr("http://caddy:2019"),
		BaseDomain:       strPtr("example.com"),
		DataDir:          strPtr("/data/hoist"),
		DefaultCPULimit:  floatPtr(1.5),
		DefaultMemoryMB:  intPtr(512),
		BuildTimeoutSec:  intPtr(300),
		HealthTimeoutSec: intPtr(60),
		FailBuildOnVulns: boolPtr(true),
		RetainedImages:   intPtr(5),
	}

	result := mergeConfigs(base, override)

	if result.SSHAddr != ":2223" {
		t.Errorf("expected SSHAddr ':2223', got '%s'", result.SSHAddr)
	}
	if result.HTTPAddr != "0.0.0.0:7575" {
		t.Errorf("expected HTTPAddr '0.0.0.0:7575', got '%s'", result.HTTPAddr)
	}
	if result.DockerHost != "tcp://docker:2375" {
		t.Errorf("expected DockerHost 'tcp://docker:2375', got '%s'", result.DockerHost)
	}
	if result.CaddyAdminURL != "http://caddy:2019" {
		t.Errorf("expected CaddyAdminURL 'http://caddy:2019', got '%s'", result.CaddyAdminURL)
	}
	if result.BaseDomain != "example.com" {
		t.Errorf("expected BaseDomain 'example.com', got '%s'", result.BaseDomain)
	}
	if result.DataDir != "/data/hoist" {
		t.Errorf("expected DataDir '/data/hoist', got '%s'", result.DataDir)
	}
	if result.DefaultCPULimit != 1.5 {
		t.Errorf("expected DefaultCPULimit 1.5, got %v", result.DefaultCPULimit)
	}
	if result.DefaultMemoryMB != 512 {
		t.Errorf("expected DefaultMemoryMB 512, got %d", result.DefaultMemoryMB)
	}
	if result.BuildTimeoutSec != 300 {
		t.Errorf("expected BuildTimeoutSec 300, got %d", result.BuildTimeoutSec)
	}
	if result.HealthTimeoutSec != 60 {
		t.Errorf("expected HealthTimeoutSec 60, got %d", result.HealthTimeoutSec)
	}
	if result.FailBuildOnVulns != true {
		t.Errorf("expected FailBuildOnVulns true, got %v", result.FailBuildOnVulns)
	}
	if result.RetainedImages != 5 {
		t.Errorf("expected RetainedImages 5, got %d", result.RetainedImages)
	}
}

func TestMergeConfigs_PartialOverride(t *testing.T) {
	base := defaults()
	override := jsonConfig{
		BaseDomain:      strPtr("example.com"),
		DefaultMemoryMB: intPtr(512),
	}

	result := mergeConfigs(base, override)

	if result.BaseDomain != "example.com" {
		t.Errorf("expected BaseDomain 'example.com', got '%s'", result.BaseDomain)
	}
	if result.DefaultMemoryMB != 512 {
		t.Errorf("expected DefaultMemoryMB 512, got %d", result.DefaultMemoryMB)
	}
	if result.SSHAddr != ":2222" {
		t.Errorf("expected SSHAddr ':2222' from defaults, got '%s'", result.SSHAddr)
	}
	if result.DataDir != "/var/lib/hoist" {
		t.Errorf("expected DataDir '/var/lib/hoist' from defaults, got '%s'", result.DataDir)
	}
}

func TestMergeConfigs_ExplicitFalseOverride(t *testing.T) {
	base := defaults()
	base.FailBuildOnVulns = true
	override := jsonConfig{
		FailBuildOnVulns: boolPtr(false),
	}

	result := mergeConfigs(base, override)

	if result.FailBuildOnVulns != false {
		t.Errorf("expected FailBuildOnVulns false (explicit override beats true base), got %v", result.FailBuildOnVulns)
	}
	if result.BaseDomain != "hoist.local" {
		t.Errorf("expected BaseDomain 'hoist.local' from base, got '%s'", result.BaseDomain)
	}
}

func TestValidateConfig_ValidConfig(t *testing.T) {
	if err := validateConfig(defaults()); err != nil {
		t.Errorf("expected no error for valid config, got %v", err)
	}
}

func TestValidateConfig_ValidAddrForms(t *testing.T) {
	addrs := []string{":2222", "127.0.0.1:2222", "0.0.0.0:7575", "[::1]:2222"}

	for _, addr := range addrs {
		cfg := defaults()
		cfg.SSHAddr = addr

		if err := validateConfig(cfg); err != nil {
			t.Errorf("expected no error for addr '%s', got %v", addr, err)
		}
	}
}

func TestValidateConfig_InvalidAddr(t *testing.T) {
	tests := []struct {
		name string
		addr string
	}{
		{"empty", ""},
		{"no port", "127.0.0.1"},
		{"no colon", "2222"},
		{"port zero", ":0"},
		{"port too high", ":70000"},
		{"port not a number", ":ssh"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := defaults()
			cfg.SSHAddr = tt.addr

			err := validateConfig(cfg)
			if err == nil {
				t.Errorf("expected error for ssh_addr '%s', got nil", tt.addr)
			} else if !strings.Contains(err.Error(), "ssh_addr") {
				t.Errorf("expected error to mention ssh_addr, got %v", err)
			}
		})
	}
}

func TestValidateConfig_InvalidHTTPAddr(t *testing.T) {
	cfg := defaults()
	cfg.HTTPAddr = "not-an-addr"

	err := validateConfig(cfg)
	if err == nil {
		t.Errorf("expected error for invalid http_addr, got nil")
	} else if !strings.Contains(err.Error(), "http_addr") {
		t.Errorf("expected error to mention http_addr, got %v", err)
	}
}

func TestValidateConfig_EmptyDockerHost(t *testing.T) {
	cfg := defaults()
	cfg.DockerHost = ""

	err := validateConfig(cfg)
	if err == nil {
		t.Errorf("expected error for empty docker_host, got nil")
	}
}

func TestValidateConfig_InvalidCaddyAdminURL(t *testing.T) {
	urls := []string{"", "not-a-url", "localhost:2019", "ftp://localhost:2019", "http://"}

	for _, u := range urls {
		cfg := defaults()
		cfg.CaddyAdminURL = u

		err := validateConfig(cfg)
		if err == nil {
			t.Errorf("expected error for caddy_admin_url '%s', got nil", u)
		}
	}
}

func TestValidateConfig_ValidBaseDomains(t *testing.T) {
	domains := []string{"hoist.local", "example.com", "localhost", "a.b.c", "my-hoist.dev"}

	for _, d := range domains {
		cfg := defaults()
		cfg.BaseDomain = d

		if err := validateConfig(cfg); err != nil {
			t.Errorf("expected no error for base_domain '%s', got %v", d, err)
		}
	}
}

func TestValidateConfig_InvalidBaseDomain(t *testing.T) {
	domains := []string{"", "-bad.com", "bad-.com", "ba..d.com", "Bad.com", ".bad.com", "bad.com.", "bad_domain.com"}

	for _, d := range domains {
		cfg := defaults()
		cfg.BaseDomain = d

		err := validateConfig(cfg)
		if err == nil {
			t.Errorf("expected error for base_domain '%s', got nil", d)
		}
	}
}

func TestValidateConfig_RelativeDataDir(t *testing.T) {
	cfg := defaults()
	cfg.DataDir = "relative/path"

	err := validateConfig(cfg)
	if err == nil {
		t.Errorf("expected error for relative data_dir, got nil")
	}
}

func TestValidateConfig_InvalidCPULimit(t *testing.T) {
	for _, cpu := range []float64{0, -0.5} {
		cfg := defaults()
		cfg.DefaultCPULimit = cpu

		err := validateConfig(cfg)
		if err == nil {
			t.Errorf("expected error for default_cpu_limit %v, got nil", cpu)
		}
	}
}

func TestValidateConfig_MemoryTooLow(t *testing.T) {
	cfg := defaults()
	cfg.DefaultMemoryMB = 4

	err := validateConfig(cfg)
	if err == nil {
		t.Errorf("expected error for default_memory_mb below Docker minimum, got nil")
	}
}

func TestValidateConfig_InvalidTimeouts(t *testing.T) {
	cfg := defaults()
	cfg.BuildTimeoutSec = 0

	if err := validateConfig(cfg); err == nil {
		t.Errorf("expected error for build_timeout_sec 0, got nil")
	}

	cfg = defaults()
	cfg.HealthTimeoutSec = -5

	if err := validateConfig(cfg); err == nil {
		t.Errorf("expected error for health_timeout_sec -5, got nil")
	}
}

func TestValidateConfig_InvalidRetainedImages(t *testing.T) {
	cfg := defaults()
	cfg.RetainedImages = 0

	err := validateConfig(cfg)
	if err == nil {
		t.Errorf("expected error for retained_images 0, got nil")
	}
}

// swapSearchPaths points config loading at hermetic temp paths for the
// duration of a test.
func swapSearchPaths(t *testing.T, paths ...string) {
	t.Helper()
	orig := searchPaths
	searchPaths = paths
	t.Cleanup(func() { searchPaths = orig })
}

func TestLoad_NoFileReturnsDefaults(t *testing.T) {
	tmpDir := t.TempDir()
	swapSearchPaths(t, filepath.Join(tmpDir, "missing.json"), filepath.Join(tmpDir, "also-missing.json"))

	cfg, err := Load()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if cfg != defaults() {
		t.Errorf("expected defaults, got %+v", cfg)
	}
}

func TestLoad_FindsFirstFile(t *testing.T) {
	tmpDir := t.TempDir()
	first := filepath.Join(tmpDir, "hoistd.json")
	second := filepath.Join(tmpDir, "other.json")

	content := `{"base_domain": "first.example.com"}`
	if err := os.WriteFile(first, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}
	content = `{"base_domain": "second.example.com"}`
	if err := os.WriteFile(second, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	swapSearchPaths(t, first, second)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if cfg.BaseDomain != "first.example.com" {
		t.Errorf("expected BaseDomain 'first.example.com' (first search path wins), got '%s'", cfg.BaseDomain)
	}
}

func TestLoad_SkipsMissingFirstPath(t *testing.T) {
	tmpDir := t.TempDir()
	missing := filepath.Join(tmpDir, "missing.json")
	second := filepath.Join(tmpDir, "hoistd.json")

	content := `{"base_domain": "example.com", "default_memory_mb": 512}`
	if err := os.WriteFile(second, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	swapSearchPaths(t, missing, second)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if cfg.BaseDomain != "example.com" {
		t.Errorf("expected BaseDomain 'example.com', got '%s'", cfg.BaseDomain)
	}
	if cfg.DefaultMemoryMB != 512 {
		t.Errorf("expected DefaultMemoryMB 512, got %d", cfg.DefaultMemoryMB)
	}
	if cfg.SSHAddr != ":2222" {
		t.Errorf("expected SSHAddr ':2222' from defaults, got '%s'", cfg.SSHAddr)
	}
}

func TestLoad_InvalidFileInSearchPath(t *testing.T) {
	tmpDir := t.TempDir()
	bad := filepath.Join(tmpDir, "hoistd.json")

	content := `{"default_memory_mb": 1}`
	if err := os.WriteFile(bad, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	swapSearchPaths(t, bad)

	_, err := Load()
	if err == nil {
		t.Errorf("expected validation error, got nil")
	}
}

func TestLoadFile_MissingFile(t *testing.T) {
	_, err := LoadFile("/nonexistent/path/hoistd.json")
	if err == nil {
		t.Errorf("expected error for missing file, got nil")
	}
}

func TestLoadFile_ExplicitZeroValueTriggersValidation(t *testing.T) {
	tmpDir := t.TempDir()
	jsonPath := filepath.Join(tmpDir, "hoistd.json")

	// "default_cpu_limit": 0 must be treated as an explicit (invalid) value,
	// not as "absent, use default".
	content := `{"default_cpu_limit": 0}`
	if err := os.WriteFile(jsonPath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	_, err := LoadFile(jsonPath)
	if err == nil {
		t.Errorf("expected validation error for explicit default_cpu_limit 0, got nil")
	}
}
