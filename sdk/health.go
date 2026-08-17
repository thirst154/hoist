package hoist

import (
	"encoding/json"
	"net/http"
	"sync"
)

type HealthChecker struct {
	checks map[string]func() error
	mu     sync.RWMutex
}

// healthChecker is initialized eagerly so that AddHealthCheck works both
// before and after Start() is called.
var healthChecker = NewHealthChecker()

func NewHealthChecker() *HealthChecker {
	return &HealthChecker{
		checks: make(map[string]func() error),
	}
}

func (h *HealthChecker) AddHealthCheck(name string, check func() error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.checks[name] = check
}

func (h *HealthChecker) Handler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		h.mu.RLock()
		checks := make(map[string]func() error)
		for k, v := range h.checks {
			checks[k] = v
		}
		h.mu.RUnlock()

		results := make(map[string]string)
		healthy := true

		for name, check := range checks {
			err := check()
			if err != nil {
				results[name] = err.Error()
				healthy = false
			} else {
				results[name] = "ok"
			}
		}

		status := "healthy"
		statusCode := http.StatusOK
		if !healthy {
			status = "unhealthy"
			statusCode = http.StatusServiceUnavailable
		}

		response := map[string]interface{}{
			"status": status,
			"checks": results,
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(statusCode)
		json.NewEncoder(w).Encode(response)
	}
}

// initHealthChecker ensures the package-level healthChecker exists without
// wiping checks that were registered before Start().
func initHealthChecker() {
	if healthChecker == nil {
		healthChecker = NewHealthChecker()
	}
}

func AddHealthCheck(name string, check func() error) {
	if healthChecker != nil {
		healthChecker.AddHealthCheck(name, check)
	}
}

func GetHealthChecker() *HealthChecker {
	return healthChecker
}
