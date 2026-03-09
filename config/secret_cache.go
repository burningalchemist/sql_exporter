package config

import (
	"sync"

	"golang.org/x/sync/singleflight"
)

var (
	secretCache  sync.Map
	secretFlight singleflight.Group
)

// ClearSecretCache drops all cached secrets, e.g. on config reload.
func ClearSecretCache() {
	secretCache.Range(func(k, _ any) bool {
		secretCache.Delete(k)
		return true
	})
}
