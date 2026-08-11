package config

import (
	"context"
	"encoding/json"
	"net/url"
	"testing"
)

// fakeSecretProvider is a mock implementation of secretProvider for testing purposes.
type fakeSecretProvider struct {
	payload map[string]string
	calls   int
}


// getDSN simulates fetching a DSN from a secret provider. It returns the JSON-encoded payload and increments the call count.
func (p *fakeSecretProvider) getDSN(_ context.Context, _ *url.URL) (string, error) {
	p.calls++
	b, err := json.Marshal(p.payload)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// TestSecretResolver_SharedFetch_DifferentKeys tests that the secretResolver fetches the secret only once when resolving different keys from the same secret.
func TestSecretResolver_SharedFetch_DifferentKeys(t *testing.T) {
	orig := secretProviders
	defer func() { secretProviders = orig }()

	fp := &fakeSecretProvider{
		payload: map[string]string{
			"writer": "postgres://writer-dsn",
			"reader": "postgres://reader-dsn",
		},
	}
	secretProviders = map[string]secretProvider{"k8ssecret": fp}

	r := &secretResolver{}
	ctx := context.Background()

	gotWriter, err := r.resolve(ctx, "k8ssecret://default/my-db-secret?key=writer")
	if err != nil {
		t.Fatalf("resolve writer: %v", err)
	}
	if gotWriter != "postgres://writer-dsn" {
		t.Fatalf("writer mismatch: got %q", gotWriter)
	}

	gotReader, err := r.resolve(ctx, "k8ssecret://default/my-db-secret?key=reader")
	if err != nil {
		t.Fatalf("resolve reader: %v", err)
	}
	if gotReader != "postgres://reader-dsn" {
		t.Fatalf("reader mismatch: got %q", gotReader)
	}

	if fp.calls != 1 {
		t.Fatalf("expected shared fetch (1 call), got %d", fp.calls)
	}
}
