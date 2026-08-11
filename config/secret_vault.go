package config

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"

	vault "github.com/hashicorp/vault/api"
)

type vaultProvider struct{}

// getDSN fetches the secret value from HashiCorp Vault KV engine.
// URL format: hashivault://mount/path?key=data_source_name&engine_version=2
func (p vaultProvider) getDSN(ctx context.Context, ref *url.URL) (string, error) {
	cfg := vault.DefaultConfig()
	if err := cfg.ReadEnvironment(); err != nil {
		return "", fmt.Errorf("unable to read Vault environment: %w", err)
	}

	client, err := vault.NewClient(cfg)
	if err != nil {
		return "", fmt.Errorf("unable to create Vault client: %w", err)
	}

	q := ref.Query()

	engineVersion := "2"
	if v := q.Get("engine_version"); v != "" {
		engineVersion = v
	}

	secretPath := ref.Host + ref.Path

	var secret *vault.KVSecret
	switch engineVersion {
	case "1":
		secret, err = client.KVv1(ref.Host).Get(ctx, ref.Path)
	default:
		secret, err = client.KVv2(ref.Host).Get(ctx, ref.Path)
	}
	if err != nil {
		return "", fmt.Errorf("unable to read Vault secret at %q: %w", secretPath, err)
	}

	// Return the raw secret payload (all keys) as JSON. Per-call ?key= selection is
	// handled by secretResolver.extractKey, since this provider's result is cached
	// (and shared) across all references to the same secret path regardless of
	// query params.
	raw := make(map[string]string, len(secret.Data))
	for k, v := range secret.Data {
		str, ok := v.(string)
		if !ok {
			return "", fmt.Errorf("value for key %q in Vault secret at %q is not a string", k, secretPath)
		}
		raw[k] = str
	}

	b, err := json.Marshal(raw)
	if err != nil {
		return "", fmt.Errorf("unable to marshal Vault secret at %q: %w", secretPath, err)
	}
	return string(b), nil
}
