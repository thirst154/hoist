package state

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	bolt "go.etcd.io/bbolt"
)

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

// PutApp inserts or updates an app. Existing apps keep their CreatedAt;
// both CreatedAt and UpdatedAt are managed by the store.
func (s *Store) PutApp(app App) error {
	return s.db.Update(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte(bucketApps))

		existing := App{}
		err := getJSON(bucket, app.Name, &existing)
		switch {
		case err == nil:
			app.CreatedAt = existing.CreatedAt
		case errors.Is(err, ErrNotFound):
			app.CreatedAt = time.Time{}
		default:
			return err
		}

		now := time.Now()
		if app.CreatedAt.IsZero() {
			app.CreatedAt = now
		}
		app.UpdatedAt = now

		return putJSON(bucket, app.Name, app)
	})
}

// GetApp returns the app with the given name. A missing app returns an
// error wrapping ErrNotFound.
func (s *Store) GetApp(name string) (App, error) {
	var app App
	err := s.db.View(func(tx *bolt.Tx) error {
		return getJSON(tx.Bucket([]byte(bucketApps)), name, &app)
	})
	if err != nil {
		return App{}, fmt.Errorf("app %q: %w", name, err)
	}
	return app, nil
}

// ListApps returns all apps sorted by name.
func (s *Store) ListApps() ([]App, error) {
	var apps []App
	err := s.db.View(func(tx *bolt.Tx) error {
		return tx.Bucket([]byte(bucketApps)).ForEach(func(k, v []byte) error {
			var app App
			if err := json.Unmarshal(v, &app); err != nil {
				return fmt.Errorf("failed to decode app %q: %w", k, err)
			}
			apps = append(apps, app)
			return nil
		})
	})
	if err != nil {
		return nil, err
	}
	return apps, nil
}
