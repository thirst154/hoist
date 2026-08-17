package state

import "time"

type SSHKey struct {
	Fingerprint string    `json:"fingerprint"`
	Key         []byte    `json:"key"`
	Comment     string    `json:"comment"`
	AddedAt     time.Time `json:"added_at"`
}
