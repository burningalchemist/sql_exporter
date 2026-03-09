package config

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/url"
	"sync"

	"golang.org/x/sync/singleflight"
)

var (
	secretCache  sync.Map
	secretFlight singleflight.Group
)

// secretProvider is the interface that all secret store backends must implement.
type secretProvider interface {
	getDSN(ctx context.Context, ref *url.URL) (string, error)
}

// secretProviders is the registry of supported secret store URL schemes.
var secretProviders = map[string]secretProvider{
	"awssecretsmanager": awsSecretsManagerProvider{},
	"gcpsecretmanager":  gcpSecretManagerProvider{},
	"hashivault":        vaultProvider{},
}

// ClearSecretCache drops all cached secrets, e.g. on config reload.
func ClearSecretCache() {
	secretCache.Range(func(k, _ any) bool {
		secretCache.Delete(k)
		return true
	})
}

// secretCacheKey returns a cache key for the secret, excluding query params
// so that multiple DSNs referencing the same secret with different keys share one fetch.
func secretCacheKey(u *url.URL) string {
	return u.Scheme + ":" + u.Host + u.Path
}

// resolveSecretDSN checks if the given value is a secret store URL and if so fetches and returns the DSN. Otherwise it
// returns the value unchanged.
func resolveSecretDSN(ctx context.Context, value string) (string, error) {
	u, err := url.Parse(value)
	if err != nil {
		return value, nil
	}

	provider, ok := secretProviders[u.Scheme]
	if !ok {
		return value, nil
	}

	cacheKey := secretCacheKey(u)

	// Check cache first
	if cached, hit := secretCache.Load(cacheKey); hit {
		slog.Debug("cache hit for secret DSN", "scheme", u.Scheme, "ref", u.Host+u.Path)
		return extractKey(cached.(string), u, value)
	}

	slog.Debug("resolving DSN from secret store", "scheme", u.Scheme, "ref", u.Host+u.Path)

	// Deduplicate concurrent fetches for the same secret.
	raw, err, _ := secretFlight.Do(cacheKey, func() (any, error) {
		if cached, hit := secretCache.Load(cacheKey); hit {
			return cached.(string), nil
		}
		result, err := provider.getDSN(ctx, u)
		if err != nil {
			return "", fmt.Errorf("failed to resolve secret %q: %w", value, err)
		}
		secretCache.Store(cacheKey, result)
		return result, nil
	})
	if err != nil {
		return "", err
	}

	return extractKey(raw.(string), u, value)
}

// extractKey pulls the appropriate key from a raw secret value (JSON or plain string).
func extractKey(raw string, u *url.URL, originalValue string) (string, error) {
	var payload map[string]string
	if jsonErr := json.Unmarshal([]byte(raw), &payload); jsonErr == nil {
		key := u.Query().Get("key")
		if key == "" {
			key = "data_source_name"
		}
		val, ok := payload[key]
		if !ok {
			return "", fmt.Errorf("key %q not found in secret %q", key, originalValue)
		}
		return val, nil
	}
	return raw, nil
}
