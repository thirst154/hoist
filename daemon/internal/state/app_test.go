package state

import (
	"errors"
	"testing"
	"time"
)

func TestPutApp_Create(t *testing.T) {
	store := newTestStore(t)

	app := App{
		Name:            "my-api",
		ActiveColor:     ColorBlue,
		LiveContainerID: "abc123",
	}

	err := store.PutApp(app)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	stored, err := store.GetApp(app.Name)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if stored.Name != "my-api" {
		t.Errorf("expected Name 'my-api', got '%s'", stored.Name)
	}
	if stored.ActiveColor != ColorBlue {
		t.Errorf("expected ActiveColor 'blue', got '%s'", stored.ActiveColor)
	}
	if stored.LiveContainerID != "abc123" {
		t.Errorf("expected LiveContainerID 'abc123', got '%s'", stored.LiveContainerID)
	}
	if stored.CreatedAt.IsZero() {
		t.Errorf("expected CreatedAt to be set by store, got zero")
	}
	if stored.UpdatedAt.IsZero() {
		t.Errorf("expected UpdatedAt to be set by store, got zero")
	}
}

func TestPutApp_UpdatePreservesCreatedAt(t *testing.T) {
	store := newTestStore(t)

	store.PutApp(App{Name: "my-api"})
	original, err := store.GetApp("my-api")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	time.Sleep(2 * time.Millisecond)

	store.PutApp(App{Name: "my-api", LiveContainerID: "new-container"})
	updated, err := store.GetApp("my-api")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if !updated.CreatedAt.Equal(original.CreatedAt) {
		t.Errorf("expected CreatedAt to be preserved, got %v -> %v", original.CreatedAt, updated.CreatedAt)
	}
}

func TestPutApp_BumpsUpdatedAt(t *testing.T) {
	store := newTestStore(t)

	store.PutApp(App{Name: "my-api"})
	original, err := store.GetApp("my-api")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	time.Sleep(2 * time.Millisecond)

	store.PutApp(App{Name: "my-api", LiveContainerID: "new-container"})
	updated, err := store.GetApp("my-api")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if !updated.UpdatedAt.After(original.UpdatedAt) {
		t.Errorf("expected UpdatedAt to advance, got %v -> %v", original.UpdatedAt, updated.UpdatedAt)
	}
}

func TestGetApp_Found(t *testing.T) {
	store := newTestStore(t)

	app := App{
		Name:             "my-api",
		ActiveColor:      ColorGreen,
		LiveContainerID:  "container-1",
		LiveDeploymentID: "7",
	}
	store.PutApp(app)

	stored, err := store.GetApp("my-api")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if stored.Name != "my-api" {
		t.Errorf("expected Name 'my-api', got '%s'", stored.Name)
	}
	if stored.ActiveColor != ColorGreen {
		t.Errorf("expected ActiveColor 'green', got '%s'", stored.ActiveColor)
	}
	if stored.LiveContainerID != "container-1" {
		t.Errorf("expected LiveContainerID 'container-1', got '%s'", stored.LiveContainerID)
	}
	if stored.LiveDeploymentID != "7" {
		t.Errorf("expected LiveDeploymentID '7', got '%s'", stored.LiveDeploymentID)
	}
}

func TestGetApp_NotFound(t *testing.T) {
	store := newTestStore(t)

	_, err := store.GetApp("missing")

	if !errors.Is(err, ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestListApps_Empty(t *testing.T) {
	store := newTestStore(t)

	apps, err := store.ListApps()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(apps) != 0 {
		t.Errorf("expected empty list, got %d apps", len(apps))
	}
}

func TestListApps_SortedByName(t *testing.T) {
	store := newTestStore(t)

	for _, name := range []string{"charlie", "alpha", "bravo"} {
		if err := store.PutApp(App{Name: name}); err != nil {
			t.Fatalf("failed to put app %q: %v", name, err)
		}
	}

	apps, err := store.ListApps()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(apps) != 3 {
		t.Fatalf("expected 3 apps, got %d", len(apps))
	}

	want := []string{"alpha", "bravo", "charlie"}
	for i, name := range want {
		if apps[i].Name != name {
			t.Errorf("expected apps[%d] to be '%s', got '%s'", i, name, apps[i].Name)
		}
	}
}
