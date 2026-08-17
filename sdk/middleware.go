package hoist

import (
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

// Request flow
// Request → RequestID → Logging → Metrics → Recovery → User Handler
//                                                              ↓
// Response ← RequestID ← Logging ← Metrics ← Recovery ← Handler response
//
// Recovery sits directly around the user handler so that a recovered panic
// is turned into a 500 through the shared responseWriter *before* the
// logging and metrics middleware record the request on their way out.


type responseWriter struct {
	http.ResponseWriter
	statusCode  int
	wroteHeader bool
}

func (rw *responseWriter) WriteHeader(code int) {
	if rw.wroteHeader {
		return
	}
	rw.wroteHeader = true
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// Unwrap returns the underlying http.ResponseWriter, so that
// http.NewResponseController features (Flush, Hijack, deadlines)
// keep working through the middleware chain.
func (rw *responseWriter) Unwrap() http.ResponseWriter {
	return rw.ResponseWriter
}

// asResponseWriter returns w unchanged if it is already a *responseWriter,
// otherwise it wraps it. This lets every middleware in the chain share a
// single status tracker.
func asResponseWriter(w http.ResponseWriter) *responseWriter {
	if rw, ok := w.(*responseWriter); ok {
		return rw
	}
	return &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}
}

func RecoveryMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rw := asResponseWriter(w)
		defer func() {
			if err := recover(); err != nil {
				logger.Error("panic recovered",
					zap.String("error", fmt.Sprintf("%v", err)),
					zap.String("path", r.URL.Path),
					zap.String("method", r.Method),
				)
				if rw.wroteHeader {
					// Headers are already on the wire and can't be changed,
					// but the request still failed: record 500 so logs and
					// metrics don't report a success.
					rw.statusCode = http.StatusInternalServerError
					return
				}
				http.Error(rw, "Internal Server Error", http.StatusInternalServerError)
			}
		}()
		next.ServeHTTP(rw, r)
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

		rw := asResponseWriter(w)
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

		rw := asResponseWriter(w)
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
		RequestIDMiddleware,
		LoggingMiddleware,
	}

	if cfg.MetricsEnabled {
		middlewares = append(middlewares, MetricsMiddleware)
	}

	// Recovery is applied last so it wraps the user handler directly.
	middlewares = append(middlewares, RecoveryMiddleware)

	return ChainMiddleware(handler, middlewares...)
}
