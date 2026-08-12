package main

import (
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

// Request flow
// Request → Recovery → RequestID → Logging → Metrics → User Handler
//                                                               ↓
// Response ← Recovery ← RequestID ← Logging ← Metrics ← Handler response


type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

func RecoveryMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				logger.Error("panic recovered",
					zap.String("error", fmt.Sprintf("%v", err)),
					zap.String("path", r.URL.Path),
					zap.String("method", r.Method),
				)
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			}
		}()
		next.ServeHTTP(w, r)
	})
}

func RequestIDMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID := r.Header.Get("X-Request-ID")
		if requestID == "" {
			requestID = uuid.New().String()
		}
		w.Header().Set("X-Request-ID", requestID)
		next.ServeHTTP(w, r)
	})
}

func LoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		
		rw := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}
		next.ServeHTTP(rw, r)
		
		duration := time.Since(start)
		
		logger.Info("request completed",
			zap.String("method", r.Method),
			zap.String("path", r.URL.Path),
			zap.Int("status", rw.statusCode),
			zap.Duration("duration", duration),
		)
	})
}

func MetricsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		
		rw := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}
		next.ServeHTTP(rw, r)
		
		duration := time.Since(start)
		
		if metrics != nil {
			metrics.RecordRequest(r.Method, r.URL.Path, rw.statusCode, duration)
		}
	})
}

func ChainMiddleware(handler http.Handler, middlewares ...func(http.Handler) http.Handler) http.Handler {
	for i := len(middlewares) - 1; i >= 0; i-- {
		handler = middlewares[i](handler)
	}
	return handler
}

func wrapHandler(handler http.Handler, cfg Config) http.Handler {
	middlewares := []func(http.Handler) http.Handler{
		RecoveryMiddleware,
		RequestIDMiddleware,
		LoggingMiddleware,
	}
	
	if cfg.MetricsEnabled {
		middlewares = append(middlewares, MetricsMiddleware)
	}
	
	return ChainMiddleware(handler, middlewares...)
}
