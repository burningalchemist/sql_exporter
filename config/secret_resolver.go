package config

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/url"

	"gocloud.dev/runtimevar"

	// Register awssecretsmanager and gcpsecretmanager URL openers.
	_ "gocloud.dev/runtimevar/awssecretsmanager"
	_ "gocloud.dev/runtimevar/gcpsecretmanager"
)

// secretPayload is the expected JSON structure of the secret value in the store.
type secretPayload struct {
	DSN string `json:"data_source_name"`
}

// resolveSecretDSN checks if the given value is a secret store URL
// (awssecretsmanager:// or gcpsecretmanager://) and if so fetches and
// returns the DSN from the secret. Otherwise it returns the value unchanged.
func resolveSecretDSN(ctx context.Context, value string) (string, error) {
	u, err := url.Parse(value)
	if err != nil || (u.Scheme != "awssecretsmanager" && u.Scheme != "gcpsecretmanager") {
		return value, nil
	}

	slog.Debug("resolving DSN from secret store", "url", value)

	v, err := runtimevar.OpenVariable(ctx, value)
	if err != nil {
		return "", fmt.Errorf("failed to open secret variable %q: %w", value, err)
	}
	defer v.Close()

	snapshot, err := v.Latest(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to fetch secret %q: %w", value, err)
	}

	// The secret value may be either a plain DSN string or a JSON object
	// with a "data_source_name" key (backward compatible with old AWS SM format).
	switch val := snapshot.Value.(type) {
	case string:
		// Try JSON first, fall back to treating it as a plain DSN string.
		var payload secretPayload
		if jsonErr := json.Unmarshal([]byte(val), &payload); jsonErr == nil && payload.DSN != "" {
			return payload.DSN, nil
		}
		return val, nil
	default:
		return "", fmt.Errorf("unexpected secret value type %T for %q", snapshot.Value, value)
	}
}
