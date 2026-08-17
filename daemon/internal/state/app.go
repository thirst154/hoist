package state

import "time"

type Color string

const (
	ColorBlue  Color = "blue"
	ColorGreen Color = "green"
)

// Other returns the opposite color. The zero value (app never deployed) maps
// to blue, the color of the first deployment.
func (c Color) Other() Color {
	if c == ColorBlue {
		return ColorGreen
	}
	return ColorBlue
}

type App struct {
	Name             string    `json:"name"` // Primary key; DNS-safe (validated by gitserver)
	ActiveColor      Color     `json:"active_color"`
	LiveContainerID  string    `json:"live_container_id"`
	LiveDeploymentID string    `json:"live_deployment_id"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

type AppConfig struct {
	Port           int    `json:"port"`
	HealthPath     string `json:"health_path"`
	LogLevel       string `json:"log_level"`
	LogFormat      string `json:"log_format"`
	MetricsEnabled bool   `json:"metrics_enabled"`
}
