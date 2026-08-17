package state

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	bolt "go.etcd.io/bbolt"
)

// openTimeout bounds how long Open waits on a locked database file. It is a
// var so tests can shrink it.
var openTimeout = 5 * time.Second

// Store wraps the BoltDB database holding all daemon state. The underlying
// db is a named field (not embedded) so callers can only go through the
// package's typed operations.
type Store struct {
	db *bolt.DB
}

// Open creates or opens the database at path, creating parent directories
// and buckets as needed.
func Open(path string) (*Store, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return nil, fmt.Errorf("failed to create data directory: %w", err)
	}

	db, err := bolt.Open(path, 0600, &bolt.Options{Timeout: openTimeout})
	if err != nil {
		return nil, fmt.Errorf("failed to open database %s: %w", path, err)
	}

	err = db.Update(func(tx *bolt.Tx) error {
		for _, name := range [][]byte{[]byte(bucketApps), []byte(bucketDeployments), []byte(bucketKeys)} {
			if _, err := tx.CreateBucketIfNotExists(name); err != nil {
				return fmt.Errorf("failed to create bucket %s: %w", name, err)
			}
		}
		return nil
	})
	if err != nil {
		db.Close()
		return nil, err
	}

	return &Store{db: db}, nil
}

// Close releases the database file lock. It is safe to call multiple times.
func (s *Store) Close() error {
	if s == nil || s.db == nil {
		return nil
	}
	err := s.db.Close()
	s.db = nil
	return err
}
