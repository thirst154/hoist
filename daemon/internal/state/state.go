// Package state persists daemon state (apps, deployments, SSH keys) in an
// embedded BoltDB database.
package state

import (
	"encoding/json"
	"errors"
	"fmt"

	bolt "go.etcd.io/bbolt"
)

// ErrNotFound is returned (wrapped) when a record does not exist. Callers
// should check with errors.Is.
var ErrNotFound = errors.New("not found")

// Bucket names. Deployments uses one nested bucket per app.
const (
	bucketApps        = "apps"
	bucketDeployments = "deployments"
	bucketKeys        = "keys"
)

// getJSON reads key from bucket into out. Returns ErrNotFound if missing.
func getJSON(b *bolt.Bucket, key string, out any) error {
	data := b.Get([]byte(key))
	if data == nil {
		return ErrNotFound
	}
	if err := json.Unmarshal(data, out); err != nil {
		return fmt.Errorf("failed to decode record %q: %w", key, err)
	}
	return nil
}

// putJSON marshals value and stores it under key in bucket.
func putJSON(b *bolt.Bucket, key string, value any) error {
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("failed to encode record %q: %w", key, err)
	}
	if err := b.Put([]byte(key), data); err != nil {
		return fmt.Errorf("failed to store record %q: %w", key, err)
	}
	return nil
}
