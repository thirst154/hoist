package main

import (
	"fmt"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type Metrics struct {
	requestsTotal   *prometheus.CounterVec
	requestDuration *prometheus.HistogramVec
	registry        *prometheus.Registry
}

var metrics *Metrics

func setupMetrics(cfg Config) (*Metrics, error) {
	if !cfg.MetricsEnabled {
		return nil, nil
	}

	registry := prometheus.NewRegistry()

	requestsTotal := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total number of HTTP requests",
		},
		[]string{"method", "path", "status"},
	)

	requestDuration := prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "HTTP request duration in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "path"},
	)

	registry.MustRegister(requestsTotal)
	registry.MustRegister(requestDuration)

	return &Metrics{
		requestsTotal:   requestsTotal,
		requestDuration: requestDuration,
		registry:        registry,
	}, nil
}

func initMetrics(cfg Config) {
	if cfg.MetricsEnabled {
		metrics, _ = setupMetrics(cfg)
	}
}

func GetMetrics() *Metrics {
	return metrics
}

func (m *Metrics) RecordRequest(method, path string, status int, duration time.Duration) {
	if m == nil {
		return
	}

	statusStr := fmt.Sprintf("%d", status)
	m.requestsTotal.WithLabelValues(method, path, statusStr).Inc()
	m.requestDuration.WithLabelValues(method, path).Observe(duration.Seconds())
}

func (m *Metrics) Handler() http.Handler {
	return promhttp.HandlerFor(m.registry, promhttp.HandlerOpts{})
}
