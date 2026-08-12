package hoist

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestSetupMetrics_Enabled(t *testing.T) {
	cfg := Config{
		MetricsEnabled: true,
	}

	m, err := setupMetrics(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if m == nil {
		t.Fatal("expected metrics to be non-nil when enabled")
	}

	if m.requestsTotal == nil {
		t.Error("expected requestsTotal counter to be non-nil")
	}

	if m.requestDuration == nil {
		t.Error("expected requestDuration histogram to be non-nil")
	}

	if m.registry == nil {
		t.Error("expected registry to be non-nil")
	}
}

func TestSetupMetrics_Disabled(t *testing.T) {
	cfg := Config{
		MetricsEnabled: false,
	}

	m, err := setupMetrics(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if m != nil {
		t.Error("expected metrics to be nil when disabled")
	}
}

func TestInitMetrics_Enabled(t *testing.T) {
	metrics = nil
	cfg := Config{
		MetricsEnabled: true,
	}

	initMetrics(cfg)

	if metrics == nil {
		t.Fatal("expected package-level metrics to be initialized")
	}
}

func TestInitMetrics_Disabled(t *testing.T) {
	metrics = nil
	cfg := Config{
		MetricsEnabled: false,
	}

	initMetrics(cfg)

	if metrics != nil {
		t.Error("expected package-level metrics to be nil when disabled")
	}
}

func TestGetMetrics_AfterInit(t *testing.T) {
	metrics = nil
	cfg := Config{
		MetricsEnabled: true,
	}

	initMetrics(cfg)

	m := GetMetrics()
	if m == nil {
		t.Fatal("expected GetMetrics to return non-nil metrics")
	}

	if m != metrics {
		t.Error("expected GetMetrics to return the package-level metrics")
	}
}

func TestGetMetrics_BeforeInit(t *testing.T) {
	metrics = nil

	m := GetMetrics()
	if m != nil {
		t.Error("expected GetMetrics to return nil before initialization")
	}
}

func TestRecordRequest(t *testing.T) {
	cfg := Config{
		MetricsEnabled: true,
	}

	m, err := setupMetrics(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	m.RecordRequest("GET", "/users", 200, 45*time.Millisecond)
	m.RecordRequest("GET", "/users", 200, 30*time.Millisecond)
	m.RecordRequest("POST", "/users", 201, 100*time.Millisecond)
	m.RecordRequest("GET", "/healthz", 200, 5*time.Millisecond)

	handler := m.Handler()
	req := httptest.NewRequest("GET", "/metrics", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	body := w.Body.String()

	if !strings.Contains(body, `http_requests_total{method="GET",path="/users",status="200"} 2`) {
		t.Error("expected counter to show 2 GET /users 200 requests")
	}

	if !strings.Contains(body, `http_requests_total{method="POST",path="/users",status="201"} 1`) {
		t.Error("expected counter to show 1 POST /users 201 request")
	}

	if !strings.Contains(body, `http_requests_total{method="GET",path="/healthz",status="200"} 1`) {
		t.Error("expected counter to show 1 GET /healthz 200 request")
	}

	if !strings.Contains(body, `http_request_duration_seconds_count{method="GET",path="/users"} 2`) {
		t.Error("expected histogram count to show 2 GET /users requests")
	}

	if !strings.Contains(body, `http_request_duration_seconds_count{method="POST",path="/users"} 1`) {
		t.Error("expected histogram count to show 1 POST /users request")
	}
}

func TestRecordRequest_DifferentStatusCodes(t *testing.T) {
	cfg := Config{
		MetricsEnabled: true,
	}

	m, err := setupMetrics(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	m.RecordRequest("GET", "/users", 200, 10*time.Millisecond)
	m.RecordRequest("GET", "/users", 404, 10*time.Millisecond)
	m.RecordRequest("GET", "/users", 500, 10*time.Millisecond)

	handler := m.Handler()
	req := httptest.NewRequest("GET", "/metrics", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	body := w.Body.String()

	if !strings.Contains(body, `http_requests_total{method="GET",path="/users",status="200"} 1`) {
		t.Error("expected counter for 200 status")
	}

	if !strings.Contains(body, `http_requests_total{method="GET",path="/users",status="404"} 1`) {
		t.Error("expected counter for 404 status")
	}

	if !strings.Contains(body, `http_requests_total{method="GET",path="/users",status="500"} 1`) {
		t.Error("expected counter for 500 status")
	}
}

func TestHandler_ReturnsValidPrometheusFormat(t *testing.T) {
	cfg := Config{
		MetricsEnabled: true,
	}

	m, err := setupMetrics(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	m.RecordRequest("GET", "/test", 200, 10*time.Millisecond)

	handler := m.Handler()
	req := httptest.NewRequest("GET", "/metrics", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	contentType := w.Header().Get("Content-Type")
	if !strings.Contains(contentType, "text/plain") {
		t.Errorf("expected Content-Type to contain text/plain, got %s", contentType)
	}

	body := w.Body.String()
	if !strings.Contains(body, "# HELP http_requests_total") {
		t.Error("expected HELP line for http_requests_total")
	}

	if !strings.Contains(body, "# TYPE http_requests_total counter") {
		t.Error("expected TYPE line for http_requests_total")
	}

	if !strings.Contains(body, "# HELP http_request_duration_seconds") {
		t.Error("expected HELP line for http_request_duration_seconds")
	}

	if !strings.Contains(body, "# TYPE http_request_duration_seconds histogram") {
		t.Error("expected TYPE line for http_request_duration_seconds")
	}
}

func TestMetrics_Isolation(t *testing.T) {
	cfg := Config{
		MetricsEnabled: true,
	}

	m1, _ := setupMetrics(cfg)
	m2, _ := setupMetrics(cfg)

	m1.RecordRequest("GET", "/test", 200, 10*time.Millisecond)

	handler1 := m1.Handler()
	req1 := httptest.NewRequest("GET", "/metrics", nil)
	w1 := httptest.NewRecorder()
	handler1.ServeHTTP(w1, req1)

	handler2 := m2.Handler()
	req2 := httptest.NewRequest("GET", "/metrics", nil)
	w2 := httptest.NewRecorder()
	handler2.ServeHTTP(w2, req2)

	body1 := w1.Body.String()
	body2 := w2.Body.String()

	if !strings.Contains(body1, `http_requests_total{method="GET",path="/test",status="200"} 1`) {
		t.Error("expected m1 to have recorded request")
	}

	if strings.Contains(body2, `http_requests_total{method="GET",path="/test"`) {
		t.Error("expected m2 to not have the request recorded in m1")
	}
}

func TestRecordRequest_NilMetrics(t *testing.T) {
	var m *Metrics

	defer func() {
		if r := recover(); r != nil {
			t.Error("expected RecordRequest to handle nil receiver gracefully")
		}
	}()

	m.RecordRequest("GET", "/test", 200, 10*time.Millisecond)
}
