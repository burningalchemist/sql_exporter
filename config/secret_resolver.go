package config

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/url"
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

	slog.Debug("resolving DSN from secret store", "scheme", u.Scheme, "ref", u.Host+u.Path)

	raw, err := provider.getDSN(ctx, u)
	if err != nil {
		return "", fmt.Errorf("failed to resolve secret %q: %w", value, err)
	}

	// If the raw value is a JSON object, extract the key specified by the "key" query param, falling back to
	// "data_source_name" for backward compatibility. If it's not JSON, return the raw value as-is.
	var payload map[string]string
	if jsonErr := json.Unmarshal([]byte(raw), &payload); jsonErr == nil {
		key := u.Query().Get("key")
		if key == "" {
			key = "data_source_name"
		}
		val, ok := payload[key]
		if !ok {
			return "", fmt.Errorf("key %q not found in secret %q", key, value)
		}
		return val, nil
	}

	return raw, nil
}
