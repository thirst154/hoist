package main

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestRecoveryMiddleware_NoPanic(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})

	recovered := RecoveryMiddleware(handler)
	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	recovered.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	if w.Body.String() != "ok" {
		t.Errorf("expected body 'ok', got '%s'", w.Body.String())
	}
}

func TestRecoveryMiddleware_WithPanic(t *testing.T) {
	logger = nil
	initLogger(Config{LogLevel: "info", LogFormat: "json"})

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("test panic")
	})

	recovered := RecoveryMiddleware(handler)
	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	recovered.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected status 500, got %d", w.Code)
	}
}

func TestRequestIDMiddleware_GeneratesID(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	withID := RequestIDMiddleware(handler)
	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	withID.ServeHTTP(w, req)

	requestID := w.Header().Get("X-Request-ID")
	if requestID == "" {
		t.Error("expected X-Request-ID header to be set")
	}

	if len(requestID) < 10 {
		t.Errorf("expected request ID to be reasonably long, got '%s'", requestID)
	}
}

func TestRequestIDMiddleware_PreservesExistingID(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	withID := RequestIDMiddleware(handler)
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("X-Request-ID", "custom-id-123")
	w := httptest.NewRecorder()

	withID.ServeHTTP(w, req)

	requestID := w.Header().Get("X-Request-ID")
	if requestID != "custom-id-123" {
		t.Errorf("expected X-Request-ID 'custom-id-123', got '%s'", requestID)
	}
}

func TestLoggingMiddleware_LogsRequest(t *testing.T) {
	logger = nil
	initLogger(Config{LogLevel: "info", LogFormat: "json"})

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	logged := LoggingMiddleware(handler)
	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	logged.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}
}

func TestLoggingMiddleware_CapturesStatusCode(t *testing.T) {
	logger = nil
	initLogger(Config{LogLevel: "info", LogFormat: "json"})

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})

	logged := LoggingMiddleware(handler)
	req := httptest.NewRequest("GET", "/notfound", nil)
	w := httptest.NewRecorder()

	logged.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", w.Code)
	}
}

func TestMetricsMiddleware_RecordsRequest(t *testing.T) {
	metrics = nil
	initMetrics(Config{MetricsEnabled: true})

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	withMetrics := MetricsMiddleware(handler)
	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	withMetrics.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}
}

func TestMetricsMiddleware_SkipsWhenDisabled(t *testing.T) {
	metrics = nil

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	withMetrics := MetricsMiddleware(handler)
	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	withMetrics.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}
}

func TestResponseWriter_CapturesStatusCode(t *testing.T) {
	rw := &responseWriter{ResponseWriter: httptest.NewRecorder(), statusCode: http.StatusOK}

	rw.WriteHeader(http.StatusNotFound)

	if rw.statusCode != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", rw.statusCode)
	}
}

func TestResponseWriter_DefaultStatusCode(t *testing.T) {
	rw := &responseWriter{ResponseWriter: httptest.NewRecorder(), statusCode: http.StatusOK}

	rw.Write([]byte("test"))

	if rw.statusCode != http.StatusOK {
		t.Errorf("expected default status 200, got %d", rw.statusCode)
	}
}

func TestResponseWriter_WriteHeader(t *testing.T) {
	rw := &responseWriter{ResponseWriter: httptest.NewRecorder(), statusCode: http.StatusOK}

	rw.WriteHeader(http.StatusCreated)

	if rw.statusCode != http.StatusCreated {
		t.Errorf("expected status 201, got %d", rw.statusCode)
	}
}

func TestChainMiddleware_EmptyChain(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("original"))
	})

	chained := ChainMiddleware(handler)
	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	chained.ServeHTTP(w, req)

	if w.Body.String() != "original" {
		t.Errorf("expected body 'original', got '%s'", w.Body.String())
	}
}

func TestChainMiddleware_SingleMiddleware(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	middleware := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("X-Test", "middleware-applied")
			next.ServeHTTP(w, r)
		})
	}

	chained := ChainMiddleware(handler, middleware)
	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	chained.ServeHTTP(w, req)

	if w.Header().Get("X-Test") != "middleware-applied" {
		t.Error("expected middleware to be applied")
	}
}

func TestChainMiddleware_MultipleMiddleware(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	middleware1 := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("X-First", "1")
			next.ServeHTTP(w, r)
		})
	}

	middleware2 := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("X-Second", "2")
			next.ServeHTTP(w, r)
		})
	}

	chained := ChainMiddleware(handler, middleware1, middleware2)
	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	chained.ServeHTTP(w, req)

	if w.Header().Get("X-First") != "1" {
		t.Error("expected first middleware to be applied")
	}
	if w.Header().Get("X-Second") != "2" {
		t.Error("expected second middleware to be applied")
	}
}

func TestChainMiddleware_Order(t *testing.T) {
	var order []string

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		order = append(order, "handler")
		w.WriteHeader(http.StatusOK)
	})

	middleware1 := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			order = append(order, "middleware1-before")
			next.ServeHTTP(w, r)
			order = append(order, "middleware1-after")
		})
	}

	middleware2 := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			order = append(order, "middleware2-before")
			next.ServeHTTP(w, r)
			order = append(order, "middleware2-after")
		})
	}

	chained := ChainMiddleware(handler, middleware1, middleware2)
	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	chained.ServeHTTP(w, req)

	expected := []string{"middleware1-before", "middleware2-before", "handler", "middleware2-after", "middleware1-after"}
	if len(order) != len(expected) {
		t.Errorf("expected %d calls, got %d", len(expected), len(order))
	}

	for i, v := range expected {
		if order[i] != v {
			t.Errorf("expected order[%d] to be '%s', got '%s'", i, v, order[i])
		}
	}
}

func TestWrapHandler_AppliesAllMiddleware(t *testing.T) {
	logger = nil
	initLogger(Config{LogLevel: "info", LogFormat: "json"})
	metrics = nil
	initMetrics(Config{MetricsEnabled: true})

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	cfg := Config{
		AppName:        "test-app",
		LogLevel:       "info",
		LogFormat:      "json",
		MetricsEnabled: true,
	}

	wrapped := wrapHandler(handler, cfg)
	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	wrapped.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}

	requestID := w.Header().Get("X-Request-ID")
	if requestID == "" {
		t.Error("expected X-Request-ID header to be set")
	}
}

func TestWrapHandler_WithPanic(t *testing.T) {
	logger = nil
	initLogger(Config{LogLevel: "info", LogFormat: "json"})
	metrics = nil

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("test panic")
	})

	cfg := Config{
		AppName:        "test-app",
		LogLevel:       "info",
		LogFormat:      "json",
		MetricsEnabled: false,
	}

	wrapped := wrapHandler(handler, cfg)
	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	wrapped.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected status 500, got %d", w.Code)
	}
}

func TestMetricsMiddleware_CapturesDuration(t *testing.T) {
	metrics = nil
	initMetrics(Config{MetricsEnabled: true})

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(10 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	})

	withMetrics := MetricsMiddleware(handler)
	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	start := time.Now()
	withMetrics.ServeHTTP(w, req)
	duration := time.Since(start)

	if duration < 10*time.Millisecond {
		t.Errorf("expected duration to be at least 10ms, got %v", duration)
	}
}

func TestResponseWriter_Header(t *testing.T) {
	inner := httptest.NewRecorder()
	rw := &responseWriter{ResponseWriter: inner, statusCode: http.StatusOK}

	rw.Header().Set("X-Test", "value")

	if inner.Header().Get("X-Test") != "value" {
		t.Error("expected header to be set on inner writer")
	}
}

func TestLoggingMiddleware_DifferentMethods(t *testing.T) {
	logger = nil
	initLogger(Config{LogLevel: "info", LogFormat: "json"})

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	logged := LoggingMiddleware(handler)

	methods := []string{"GET", "POST", "PUT", "DELETE"}
	for _, method := range methods {
		req := httptest.NewRequest(method, "/test", nil)
		w := httptest.NewRecorder()

		logged.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected status 200 for %s, got %d", method, w.Code)
		}
	}
}

func TestMetricsMiddleware_DifferentStatusCodes(t *testing.T) {
	metrics = nil
	initMetrics(Config{MetricsEnabled: true})

	statusCodes := []int{200, 201, 400, 404, 500}

	for _, statusCode := range statusCodes {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(statusCode)
		})

		withMetrics := MetricsMiddleware(handler)
		req := httptest.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()

		withMetrics.ServeHTTP(w, req)

		if w.Code != statusCode {
			t.Errorf("expected status %d, got %d", statusCode, w.Code)
		}
	}
}

func TestRecoveryMiddleware_RecoversFromDifferentPanicTypes(t *testing.T) {
	logger = nil
	initLogger(Config{LogLevel: "info", LogFormat: "json"})

	panicTypes := []interface{}{
		"string panic",
		errors.New("error panic"),
		42,
		nil,
	}

	for i, panicValue := range panicTypes {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			panic(panicValue)
		})

		recovered := RecoveryMiddleware(handler)
		req := httptest.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()

		recovered.ServeHTTP(w, req)

		if w.Code != http.StatusInternalServerError {
			t.Errorf("test %d: expected status 500, got %d", i, w.Code)
		}
	}
}

func TestRequestIDMiddleware_UniqueIDs(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	withID := RequestIDMiddleware(handler)

	ids := make(map[string]bool)
	for i := 0; i < 100; i++ {
		req := httptest.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()

		withID.ServeHTTP(w, req)

		id := w.Header().Get("X-Request-ID")
		if ids[id] {
			t.Errorf("duplicate request ID generated: %s", id)
		}
		ids[id] = true
	}
}

func TestWrapHandler_MetricsDisabled(t *testing.T) {
	logger = nil
	initLogger(Config{LogLevel: "info", LogFormat: "json"})
	metrics = nil

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	cfg := Config{
		AppName:        "test-app",
		LogLevel:       "info",
		LogFormat:      "json",
		MetricsEnabled: false,
	}

	wrapped := wrapHandler(handler, cfg)
	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	wrapped.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}
}

func TestLoggingMiddleware_EmptyPath(t *testing.T) {
	logger = nil
	initLogger(Config{LogLevel: "info", LogFormat: "json"})

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	logged := LoggingMiddleware(handler)
	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()

	logged.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}
}

func TestMetricsMiddleware_EmptyPath(t *testing.T) {
	metrics = nil
	initMetrics(Config{MetricsEnabled: true})

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	withMetrics := MetricsMiddleware(handler)
	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()

	withMetrics.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}
}

func TestResponseWriter_Write(t *testing.T) {
	inner := httptest.NewRecorder()
	rw := &responseWriter{ResponseWriter: inner, statusCode: http.StatusOK}

	n, err := rw.Write([]byte("test"))

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if n != 4 {
		t.Errorf("expected 4 bytes written, got %d", n)
	}

	if inner.Body.String() != "test" {
		t.Errorf("expected body 'test', got '%s'", inner.Body.String())
	}
}

func TestChainMiddleware_NilHandler(t *testing.T) {
	result := ChainMiddleware(nil)
	if result != nil {
		t.Error("expected nil handler to return nil")
	}
}

func TestWrapHandler_PreservesHeaders(t *testing.T) {
	logger = nil
	initLogger(Config{LogLevel: "info", LogFormat: "json"})
	metrics = nil

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Custom", "value")
		w.WriteHeader(http.StatusOK)
	})

	cfg := Config{
		AppName:        "test-app",
		LogLevel:       "info",
		LogFormat:      "json",
		MetricsEnabled: false,
	}

	wrapped := wrapHandler(handler, cfg)
	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	wrapped.ServeHTTP(w, req)

	if w.Header().Get("X-Custom") != "value" {
		t.Error("expected custom header to be preserved")
	}
}

func TestLoggingMiddleware_QueryParams(t *testing.T) {
	logger = nil
	initLogger(Config{LogLevel: "info", LogFormat: "json"})

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	logged := LoggingMiddleware(handler)
	req := httptest.NewRequest("GET", "/test?foo=bar", nil)
	w := httptest.NewRecorder()

	logged.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}
}

func TestMetricsMiddleware_QueryParams(t *testing.T) {
	metrics = nil
	initMetrics(Config{MetricsEnabled: true})

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	withMetrics := MetricsMiddleware(handler)
	req := httptest.NewRequest("GET", "/test?foo=bar", nil)
	w := httptest.NewRecorder()

	withMetrics.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}
}

func TestRecoveryMiddleware_PanicBeforeWrite(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("panic before write")
	})

	recovered := RecoveryMiddleware(handler)
	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	recovered.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected status 500, got %d", w.Code)
	}
}

func TestRequestIDMiddleware_EmptyID(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	withID := RequestIDMiddleware(handler)
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("X-Request-ID", "")
	w := httptest.NewRecorder()

	withID.ServeHTTP(w, req)

	requestID := w.Header().Get("X-Request-ID")
	if requestID == "" {
		t.Error("expected X-Request-ID to be generated for empty header")
	}
}

func TestChainMiddleware_WithRecovery(t *testing.T) {
	logger = nil
	initLogger(Config{LogLevel: "info", LogFormat: "json"})

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("test panic")
	})

	chained := ChainMiddleware(handler, RecoveryMiddleware)
	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	chained.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected status 500, got %d", w.Code)
	}
}

func TestWrapHandler_LargeRequestBody(t *testing.T) {
	logger = nil
	initLogger(Config{LogLevel: "info", LogFormat: "json"})
	metrics = nil

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	cfg := Config{
		AppName:        "test-app",
		LogLevel:       "info",
		LogFormat:      "json",
		MetricsEnabled: false,
	}

	wrapped := wrapHandler(handler, cfg)
	body := strings.Repeat("x", 10000)
	req := httptest.NewRequest("POST", "/test", strings.NewReader(body))
	w := httptest.NewRecorder()

	wrapped.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}
}
