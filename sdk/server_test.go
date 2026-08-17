package hoist

import (
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"syscall"
	"testing"
	"time"
)

func TestSetupRoutes_MountsHealthEndpoint(t *testing.T) {
	healthChecker = nil
	initHealthChecker()

	userHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("user handler"))
	})

	cfg := Config{
		HealthPath:     "/healthz",
		MetricsEnabled: false,
	}

	handler := setupRoutes(userHandler, cfg)

	req := httptest.NewRequest("GET", "/healthz", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200 for health endpoint, got %d", w.Code)
	}
}

func TestSetupRoutes_MountsMetricsEndpoint(t *testing.T) {
	metrics = nil
	initMetrics(Config{MetricsEnabled: true})

	userHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	cfg := Config{
		HealthPath:     "/healthz",
		MetricsEnabled: true,
	}

	handler := setupRoutes(userHandler, cfg)

	req := httptest.NewRequest("GET", "/metrics", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200 for metrics endpoint, got %d", w.Code)
	}
}

func TestSetupRoutes_UserHandlerStillWorks(t *testing.T) {
	healthChecker = nil
	initHealthChecker()

	userHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("user response"))
	})

	cfg := Config{
		HealthPath:     "/healthz",
		MetricsEnabled: false,
	}

	handler := setupRoutes(userHandler, cfg)

	req := httptest.NewRequest("GET", "/custom", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Body.String() != "user response" {
		t.Errorf("expected 'user response', got '%s'", w.Body.String())
	}
}

func TestSetupRoutes_HealthPathFromConfig(t *testing.T) {
	healthChecker = nil
	initHealthChecker()

	userHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	cfg := Config{
		HealthPath:     "/custom-health",
		MetricsEnabled: false,
	}

	handler := setupRoutes(userHandler, cfg)

	req := httptest.NewRequest("GET", "/custom-health", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200 for custom health path, got %d", w.Code)
	}
}

func TestSetupRoutes_MetricsDisabled(t *testing.T) {
	metrics = nil

	userHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})

	cfg := Config{
		HealthPath:     "/healthz",
		MetricsEnabled: false,
	}

	handler := setupRoutes(userHandler, cfg)

	req := httptest.NewRequest("GET", "/metrics", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status 404 when metrics disabled, got %d", w.Code)
	}
}

func TestStartServer_CreatesServer(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	cfg := Config{
		Port: 8081,
	}

	server := startServer(handler, cfg)
	t.Cleanup(func() { server.Close() })

	if server == nil {
		t.Fatal("expected server to be non-nil")
	}

	if server.Addr != ":8081" {
		t.Errorf("expected addr ':8081', got '%s'", server.Addr)
	}

	if server.ReadTimeout != 15*time.Second {
		t.Errorf("expected ReadTimeout 15s, got %v", server.ReadTimeout)
	}

	if server.WriteTimeout != 15*time.Second {
		t.Errorf("expected WriteTimeout 15s, got %v", server.WriteTimeout)
	}

	if server.IdleTimeout != 60*time.Second {
		t.Errorf("expected IdleTimeout 60s, got %v", server.IdleTimeout)
	}
}

func TestStartServer_DifferentPort(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	cfg := Config{
		Port: 3000,
	}

	server := startServer(handler, cfg)
	t.Cleanup(func() { server.Close() })

	if server.Addr != ":3000" {
		t.Errorf("expected addr ':3000', got '%s'", server.Addr)
	}
}

func TestGracefulShutdown_ShutsDownServer(t *testing.T) {
	logger = nil
	initLogger(Config{LogLevel: "info", LogFormat: "json"})

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	})

	server := &http.Server{
		Addr:    ":8082",
		Handler: handler,
	}

	go server.ListenAndServe()
	time.Sleep(50 * time.Millisecond)

	stop := make(chan os.Signal, 1)
	done := make(chan bool, 1)

	go func() {
		gracefulShutdown(server, stop)
		done <- true
	}()

	stop <- syscall.SIGTERM

	select {
	case <-done:
		// Success
	case <-time.After(2 * time.Second):
		t.Fatal("shutdown took too long")
	}
}

func TestGracefulShutdown_DrainsConnections(t *testing.T) {
	logger = nil
	initLogger(Config{LogLevel: "info", LogFormat: "json"})

	requestStarted := make(chan bool)
	requestCompleted := make(chan bool)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestStarted <- true
		time.Sleep(200 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
		requestCompleted <- true
	})

	server := &http.Server{
		Addr:    ":8083",
		Handler: handler,
	}

	go server.ListenAndServe()
	time.Sleep(50 * time.Millisecond)

	stop := make(chan os.Signal, 1)
	go gracefulShutdown(server, stop)

	go func() {
		resp, err := http.Get("http://localhost:8083/test")
		if err == nil {
			resp.Body.Close()
		}
	}()

	<-requestStarted
	stop <- syscall.SIGTERM

	select {
	case <-requestCompleted:
		// Request completed before shutdown
	case <-time.After(2 * time.Second):
		t.Fatal("request did not complete during graceful shutdown")
	}
}

func TestStart_Integration(t *testing.T) {
	os.Clearenv()

	tmpDir := t.TempDir()
	origDir, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(origDir)

	configJSON := `{
		"app_name": "test-app",
		"port": 8084,
		"log_level": "info",
		"log_format": "json",
		"metrics_enabled": true,
		"health_path": "/healthz"
	}`
	os.WriteFile("hoist.json", []byte(configJSON), 0644)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("test response"))
	})

	done := make(chan bool, 1)
	go func() {
		Start(handler)
		done <- true
	}()

	time.Sleep(100 * time.Millisecond)

	resp, err := http.Get("http://localhost:8084/healthz")
	if err != nil {
		t.Fatalf("failed to hit health endpoint: %v", err)
	}
	resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status 200 for health endpoint, got %d", resp.StatusCode)
	}

	resp, err = http.Get("http://localhost:8084/metrics")
	if err != nil {
		t.Fatalf("failed to hit metrics endpoint: %v", err)
	}
	resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status 200 for metrics endpoint, got %d", resp.StatusCode)
	}

	resp, err = http.Get("http://localhost:8084/test")
	if err != nil {
		t.Fatalf("failed to hit user endpoint: %v", err)
	}
	resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status 200 for user endpoint, got %d", resp.StatusCode)
	}

	syscall.Kill(syscall.Getpid(), syscall.SIGTERM)

	select {
	case <-done:
		// Success
	case <-time.After(2 * time.Second):
		t.Fatal("server did not shut down")
	}
}

func TestStart_InvalidConfig(t *testing.T) {
	os.Clearenv()

	tmpDir := t.TempDir()
	origDir, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(origDir)

	configJSON := `{
		"port": 70000
	}`
	os.WriteFile("hoist.json", []byte(configJSON), 0644)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	defer func() {
		if r := recover(); r == nil {
			t.Error("expected Start to panic on invalid config")
		}
	}()

	Start(handler)
}

func TestSetupRoutes_HealthWithCustomChecks(t *testing.T) {
	healthChecker = nil
	initHealthChecker()

	AddHealthCheck("database", func() error {
		return nil
	})

	userHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	cfg := Config{
		HealthPath:     "/healthz",
		MetricsEnabled: false,
	}

	handler := setupRoutes(userHandler, cfg)

	req := httptest.NewRequest("GET", "/healthz", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}
}

func TestSetupRoutes_MetricsWithRecordedData(t *testing.T) {
	metrics = nil
	initMetrics(Config{MetricsEnabled: true})

	metrics.RecordRequest("GET", "/test", 200, 10*time.Millisecond)

	userHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	cfg := Config{
		HealthPath:     "/healthz",
		MetricsEnabled: true,
	}

	handler := setupRoutes(userHandler, cfg)

	req := httptest.NewRequest("GET", "/metrics", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	body := w.Body.String()
	if body == "" {
		t.Error("expected metrics response to contain data")
	}
}

func TestStartServer_HandlerIsSet(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	cfg := Config{
		Port: 8085,
	}

	server := startServer(handler, cfg)
	t.Cleanup(func() { server.Close() })

	if server.Handler == nil {
		t.Error("expected server handler to be set")
	}
}

func TestGracefulShutdown_ContextTimeout(t *testing.T) {
	logger = nil
	initLogger(Config{LogLevel: "info", LogFormat: "json"})

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(10 * time.Second)
		w.WriteHeader(http.StatusOK)
	})

	server := &http.Server{
		Addr:    ":8086",
		Handler: handler,
	}

	go server.ListenAndServe()
	time.Sleep(50 * time.Millisecond)

	stop := make(chan os.Signal, 1)
	done := make(chan bool, 1)

	go func() {
		gracefulShutdown(server, stop)
		done <- true
	}()

	stop <- syscall.SIGTERM

	select {
	case <-done:
		// Shutdown completed (with or without timeout)
	case <-time.After(35 * time.Second):
		t.Fatal("shutdown took too long, context timeout not working")
	}
}

func TestStart_MultipleHealthChecks(t *testing.T) {
	os.Clearenv()

	tmpDir := t.TempDir()
	origDir, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(origDir)

	configJSON := `{
		"app_name": "test-app",
		"port": 8087,
		"log_level": "info",
		"log_format": "json",
		"metrics_enabled": false,
		"health_path": "/healthz"
	}`
	os.WriteFile("hoist.json", []byte(configJSON), 0644)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	done := make(chan bool, 1)
	go func() {
		Start(handler)
		done <- true
	}()

	time.Sleep(100 * time.Millisecond)

	AddHealthCheck("db", func() error { return nil })
	AddHealthCheck("cache", func() error { return nil })

	resp, err := http.Get("http://localhost:8087/healthz")
	if err != nil {
		t.Fatalf("failed to hit health endpoint: %v", err)
	}
	resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}

	syscall.Kill(syscall.Getpid(), syscall.SIGTERM)

	select {
	case <-done:
		// Success
	case <-time.After(2 * time.Second):
		t.Fatal("server did not shut down")
	}
}

func TestSetupRoutes_EmptyHealthPath(t *testing.T) {
	healthChecker = nil
	initHealthChecker()

	userHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	cfg := Config{
		HealthPath:     "",
		MetricsEnabled: false,
	}

	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic for empty health path")
		}
	}()

	setupRoutes(userHandler, cfg)
}

func TestStartServer_ZeroPort(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	cfg := Config{
		Port: 0,
	}

	server := startServer(handler, cfg)
	t.Cleanup(func() { server.Close() })

	if server.Addr != ":0" {
		t.Errorf("expected addr ':0', got '%s'", server.Addr)
	}
}

func TestStartServer_PortInUse(t *testing.T) {
	// Occupy an ephemeral port
	occupied, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatalf("failed to occupy port: %v", err)
	}
	defer occupied.Close()
	port := occupied.Addr().(*net.TCPAddr).Port

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	cfg := Config{
		Port: port,
	}

	defer func() {
		if r := recover(); r == nil {
			t.Error("expected startServer to panic when the port is already in use")
		}
	}()

	startServer(handler, cfg)
}
