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

// rawSecretProvider returns a fixed secret string as stored, the way the AWS and GCP providers do.
type rawSecretProvider struct {
	raw string
}

func (p rawSecretProvider) getDSN(_ context.Context, _ *url.URL) (string, error) {
	return p.raw, nil
}

// TestSecretResolver_JSONSecretValueTypes tests that a JSON secret is read as a JSON object whatever the type of its
// other values, so that a secret such as a database secret with a numeric "port" still honours the key and template
// of the reference.
func TestSecretResolver_JSONSecretValueTypes(t *testing.T) {
	const dbSecret = `{"engine":"postgres","host":"db.example.com","port":5432,"ssl":true,` +
		`"data_source_name":"postgres://user:pass@db.example.com:5432/app","reader":"postgres://reader@db.example.com/app"}`

	tests := []struct {
		name    string
		raw     string
		ref     string
		want    string
		wantErr bool
	}{
		{
			name: "default key beside a numeric and a boolean value",
			raw:  dbSecret,
			ref:  "awssecretsmanager://prod/db?region=eu-west-1",
			want: "postgres://user:pass@db.example.com:5432/app",
		},
		{
			name: "template beside a numeric value",
			raw:  dbSecret,
			ref:  "awssecretsmanager://prod/db?key=reader&template=DSN_VALUE%3Fsslmode%3Drequire",
			want: "postgres://reader@db.example.com/app?sslmode=require",
		},
		{
			name:    "missing key beside a numeric value",
			raw:     dbSecret,
			ref:     "awssecretsmanager://prod/db?key=writer",
			wantErr: true,
		},
		{
			name:    "selected value is a number",
			raw:     dbSecret,
			ref:     "awssecretsmanager://prod/db?key=port",
			wantErr: true,
		},
		{
			name: "plain string secret",
			raw:  "postgres://user:pass@db.example.com:5432/app",
			ref:  "awssecretsmanager://prod/db",
			want: "postgres://user:pass@db.example.com:5432/app",
		},
	}

	orig := secretProviders
	defer func() { secretProviders = orig }()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			secretProviders = map[string]secretProvider{"awssecretsmanager": rawSecretProvider{raw: tt.raw}}

			got, err := (&secretResolver{}).resolve(context.Background(), tt.ref)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected an error, got %q", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("resolve: %v", err)
			}
			if got != tt.want {
				t.Fatalf("got %q, want %q", got, tt.want)
			}
		})
	}
}
