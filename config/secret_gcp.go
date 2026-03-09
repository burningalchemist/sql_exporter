package config

import (
	"context"
	"fmt"
	"net/url"

	secretmanager "cloud.google.com/go/secretmanager/apiv1"
	"cloud.google.com/go/secretmanager/apiv1/secretmanagerpb"
)

type gcpSecretManagerProvider struct{}

// getDSN fetches the secret value from GCP Secret Manager.
// URL format: gcpsecretmanager://projects/my-project/secrets/my-secret
func (p gcpSecretManagerProvider) getDSN(ctx context.Context, ref *url.URL) (string, error) {
	client, err := secretmanager.NewClient(ctx)
	if err != nil {
		return "", fmt.Errorf("unable to create GCP Secret Manager client: %w", err)
	}
	defer client.Close()

	// Reconstruct the full resource name from host+path and append /versions/latest.
	secretName := ref.Host + ref.Path + "/versions/latest"

	result, err := client.AccessSecretVersion(ctx, &secretmanagerpb.AccessSecretVersionRequest{
		Name: secretName,
	})
	if err != nil {
		return "", fmt.Errorf("unable to access secret version: %w", err)
	}

	return string(result.Payload.Data), nil
}
