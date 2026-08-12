package hoist

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
)

func TestNewHealthChecker(t *testing.T) {
	hc := NewHealthChecker()

	if hc == nil {
		t.Fatal("expected health checker to be non-nil")
	}

	if hc.checks == nil {
		t.Error("expected checks map to be initialized")
	}
}

func TestAddHealthCheck_SingleCheck(t *testing.T) {
	hc := NewHealthChecker()

	hc.AddHealthCheck("database", func() error {
		return nil
	})

	if len(hc.checks) != 1 {
		t.Errorf("expected 1 check, got %d", len(hc.checks))
	}

	if _, ok := hc.checks["database"]; !ok {
		t.Error("expected 'database' check to be registered")
	}
}

func TestAddHealthCheck_MultipleChecks(t *testing.T) {
	hc := NewHealthChecker()

	hc.AddHealthCheck("database", func() error { return nil })
	hc.AddHealthCheck("redis", func() error { return nil })
	hc.AddHealthCheck("cache", func() error { return nil })

	if len(hc.checks) != 3 {
		t.Errorf("expected 3 checks, got %d", len(hc.checks))
	}
}

func TestAddHealthCheck_OverwriteExisting(t *testing.T) {
	hc := NewHealthChecker()

	hc.AddHealthCheck("database", func() error { return nil })
	hc.AddHealthCheck("database", func() error { return errors.New("fail") })

	if len(hc.checks) != 1 {
		t.Errorf("expected 1 check after overwrite, got %d", len(hc.checks))
	}
}

func TestAddHealthCheck_Concurrent(t *testing.T) {
	hc := NewHealthChecker()

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			name := string(rune('a' + i%26))
			hc.AddHealthCheck(name, func() error { return nil })
		}(i)
	}
	wg.Wait()

	hc.mu.RLock()
	count := len(hc.checks)
	hc.mu.RUnlock()

	if count == 0 {
		t.Error("expected at least some checks to be registered")
	}
}

func TestHandler_AllChecksPass(t *testing.T) {
	hc := NewHealthChecker()
	hc.AddHealthCheck("database", func() error { return nil })
	hc.AddHealthCheck("redis", func() error { return nil })

	handler := hc.Handler()
	req := httptest.NewRequest("GET", "/healthz", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	if err != nil {
		t.Fatalf("failed to parse response JSON: %v", err)
	}

	if response["status"] != "healthy" {
		t.Errorf("expected status 'healthy', got '%v'", response["status"])
	}

	checks := response["checks"].(map[string]interface{})
	if checks["database"] != "ok" {
		t.Errorf("expected database check to be 'ok', got '%v'", checks["database"])
	}
	if checks["redis"] != "ok" {
		t.Errorf("expected redis check to be 'ok', got '%v'", checks["redis"])
	}
}

func TestHandler_SomeChecksFail(t *testing.T) {
	hc := NewHealthChecker()
	hc.AddHealthCheck("database", func() error { return nil })
	hc.AddHealthCheck("redis", func() error { return errors.New("connection refused") })

	handler := hc.Handler()
	req := httptest.NewRequest("GET", "/healthz", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("expected status 503, got %d", w.Code)
	}

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	if err != nil {
		t.Fatalf("failed to parse response JSON: %v", err)
	}

	if response["status"] != "unhealthy" {
		t.Errorf("expected status 'unhealthy', got '%v'", response["status"])
	}

	checks := response["checks"].(map[string]interface{})
	if checks["database"] != "ok" {
		t.Errorf("expected database check to be 'ok', got '%v'", checks["database"])
	}
	if checks["redis"] != "connection refused" {
		t.Errorf("expected redis check to be 'connection refused', got '%v'", checks["redis"])
	}
}

func TestHandler_AllChecksFail(t *testing.T) {
	hc := NewHealthChecker()
	hc.AddHealthCheck("database", func() error { return errors.New("timeout") })
	hc.AddHealthCheck("redis", func() error { return errors.New("connection refused") })

	handler := hc.Handler()
	req := httptest.NewRequest("GET", "/healthz", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("expected status 503, got %d", w.Code)
	}

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	if err != nil {
		t.Fatalf("failed to parse response JSON: %v", err)
	}

	if response["status"] != "unhealthy" {
		t.Errorf("expected status 'unhealthy', got '%v'", response["status"])
	}
}

func TestHandler_NoChecksRegistered(t *testing.T) {
	hc := NewHealthChecker()

	handler := hc.Handler()
	req := httptest.NewRequest("GET", "/healthz", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200 when no checks registered, got %d", w.Code)
	}

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	if err != nil {
		t.Fatalf("failed to parse response JSON: %v", err)
	}

	if response["status"] != "healthy" {
		t.Errorf("expected status 'healthy', got '%v'", response["status"])
	}
}

func TestHandler_ContentType(t *testing.T) {
	hc := NewHealthChecker()
	hc.AddHealthCheck("test", func() error { return nil })

	handler := hc.Handler()
	req := httptest.NewRequest("GET", "/healthz", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	contentType := w.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("expected Content-Type 'application/json', got '%s'", contentType)
	}
}

func TestInitHealthChecker(t *testing.T) {
	healthChecker = nil

	initHealthChecker()

	if healthChecker == nil {
		t.Fatal("expected package-level healthChecker to be initialized")
	}
}

func TestGetHealthChecker_AfterInit(t *testing.T) {
	healthChecker = nil
	initHealthChecker()

	hc := GetHealthChecker()

	if hc == nil {
		t.Fatal("expected GetHealthChecker to return non-nil")
	}

	if hc != healthChecker {
		t.Error("expected GetHealthChecker to return the package-level healthChecker")
	}
}

func TestGetHealthChecker_BeforeInit(t *testing.T) {
	healthChecker = nil

	hc := GetHealthChecker()

	if hc != nil {
		t.Error("expected GetHealthChecker to return nil before initialization")
	}
}

func TestPublicAddHealthCheck(t *testing.T) {
	healthChecker = nil
	initHealthChecker()

	AddHealthCheck("database", func() error { return nil })

	hc := GetHealthChecker()
	if len(hc.checks) != 1 {
		t.Errorf("expected 1 check, got %d", len(hc.checks))
	}
}

func TestPublicAddHealthCheck_BeforeInit(t *testing.T) {
	healthChecker = nil

	defer func() {
		if r := recover(); r != nil {
			t.Error("expected AddHealthCheck to handle nil healthChecker gracefully")
		}
	}()

	AddHealthCheck("database", func() error { return nil })
}

func TestHandler_CheckReturnsNilError(t *testing.T) {
	hc := NewHealthChecker()
	hc.AddHealthCheck("test", func() error { return nil })

	handler := hc.Handler()
	req := httptest.NewRequest("GET", "/healthz", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)

	checks := response["checks"].(map[string]interface{})
	if checks["test"] != "ok" {
		t.Errorf("expected check to be 'ok', got '%v'", checks["test"])
	}
}

func TestHandler_CheckReturnsEmptyError(t *testing.T) {
	hc := NewHealthChecker()
	hc.AddHealthCheck("test", func() error { return errors.New("") })

	handler := hc.Handler()
	req := httptest.NewRequest("GET", "/healthz", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("expected status 503 for empty error, got %d", w.Code)
	}
}
