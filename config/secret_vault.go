package config

import (
	"context"
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

	// key query param specifies which field to extract, defaults to "data_source_name".
	key := q.Get("key")
	if key == "" {
		key = "data_source_name"
	}

	val, ok := secret.Data[key]
	if !ok {
		return "", fmt.Errorf("key %q not found in Vault secret at %q", key, secretPath)
	}

	str, ok := val.(string)
	if !ok {
		return "", fmt.Errorf("key %q in Vault secret at %q is not a string", key, secretPath)
	}

	return str, nil
}
