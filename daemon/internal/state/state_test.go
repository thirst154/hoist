package state

import (
	"errors"
	"path/filepath"
	"testing"
	"time"

	bolt "go.etcd.io/bbolt"
)

// newTestStore opens a Store in a temp directory and closes it on cleanup.
func newTestStore(t *testing.T) *Store {
	t.Helper()
	store, err := Open(filepath.Join(t.TempDir(), "hoist.db"))
	if err != nil {
		t.Fatalf("failed to open test store: %v", err)
	}
	t.Cleanup(func() { store.Close() })
	return store
}

func TestOpen_CreatesBuckets(t *testing.T) {
	store := newTestStore(t)

	err := store.db.View(func(tx *bolt.Tx) error {
		for _, name := range []string{bucketApps, bucketDeployments, bucketKeys} {
			if tx.Bucket([]byte(name)) == nil {
				t.Errorf("expected bucket %q to exist", name)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("view failed: %v", err)
	}
}

func TestOpen_ExistingDatabase(t *testing.T) {
	path := filepath.Join(t.TempDir(), "hoist.db")

	store, err := Open(path)
	if err != nil {
		t.Fatalf("first open failed: %v", err)
	}
	store.Close()

	store2, err := Open(path)
	if err != nil {
		t.Fatalf("reopen failed: %v", err)
	}
	defer store2.Close()

	// Buckets must survive the reopen.
	err = store2.db.View(func(tx *bolt.Tx) error {
		for _, name := range []string{bucketApps, bucketDeployments, bucketKeys} {
			if tx.Bucket([]byte(name)) == nil {
				t.Errorf("expected bucket %q to exist after reopen", name)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("view failed: %v", err)
	}
}

func TestOpen_CreatesParentDirectories(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested", "dir", "hoist.db")

	store, err := Open(path)
	if err != nil {
		t.Fatalf("expected open with missing parent dirs to succeed, got %v", err)
	}
	defer store.Close()
}

func TestOpen_LockedDatabase(t *testing.T) {
	path := filepath.Join(t.TempDir(), "hoist.db")

	store, err := Open(path)
	if err != nil {
		t.Fatalf("first open failed: %v", err)
	}
	defer store.Close()

	// Shrink the lock timeout so the test fails fast instead of waiting 5s.
	orig := openTimeout
	openTimeout = 100 * time.Millisecond
	t.Cleanup(func() { openTimeout = orig })

	// The file is already locked; a second open must fail rather than block.
	if _, err := Open(path); err == nil {
		t.Errorf("expected error opening locked database, got nil")
	}
}

func TestClose_Twice(t *testing.T) {
	store, err := Open(filepath.Join(t.TempDir(), "hoist.db"))
	if err != nil {
		t.Fatalf("open failed: %v", err)
	}

	if err := store.Close(); err != nil {
		t.Errorf("first close failed: %v", err)
	}
	if err := store.Close(); err != nil {
		t.Errorf("second close should be a no-op, got %v", err)
	}
}

func TestErrNotFound_IsSentinel(t *testing.T) {
	if !errors.Is(ErrNotFound, ErrNotFound) {
		t.Errorf("ErrNotFound must be usable with errors.Is")
	}
}
