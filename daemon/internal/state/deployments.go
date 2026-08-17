package state

import "time"

type DeploymentStatus string

const (
	StatusBuilding DeploymentStatus = "building"
	StatusStarting DeploymentStatus = "starting"
	StatusLive     DeploymentStatus = "live"
	StatusFailed   DeploymentStatus = "failed"
	StatusReplaced DeploymentStatus = "replaced"
)

type Deployment struct {
	ID           string           `json:"id"`
	AppName      string           `json:"app_name"`
	SHA          string           `json:"sha"`
	Status       DeploymentStatus `json:"status"`
	Color        Color            `json:"color"`
	ContainerID  string           `json:"container_id"`
	ImageTag     string           `json:"image_tag"`
	Config       AppConfig        `json:"config"`
	BuildLogPath string           `json:"build_log_path"`
	Error        string           `json:"error"`
	CreatedAt    time.Time        `json:"created_at"`
	UpdatedAt    time.Time        `json:"updated_at"`
}
