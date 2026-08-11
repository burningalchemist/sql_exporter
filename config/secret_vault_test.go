package config

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	vault "github.com/hashicorp/vault/api"
)

// newVaultTestServer starts an httptest server that serves a fixed KV v2 response for any path, 
// mimicking Vault's /v1/<mount>/data/<path> shape.
func newVaultTestServer(t *testing.T, data map[string]any) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		resp := map[string]any{
			"data": map[string]any{
				"data": data,
				"metadata": map[string]any{
					"version": 1,
				},
			},
		}
		_ = json.NewEncoder(w).Encode(resp)
	}))
}

// newVaultTestServerV1 mimics a KV v1 response shape (no nested "data").
func newVaultTestServerV1(t *testing.T, data map[string]any) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		resp := map[string]any{
			"data": data,
		}
		_ = json.NewEncoder(w).Encode(resp)
	}))
}

// setVaultAddrEnv sets the VAULT_ADDR and VAULT_TOKEN environment variables for testing.
func setVaultAddrEnv(t *testing.T, addr string) {
	t.Helper()
	t.Setenv("VAULT_ADDR", addr)
	// Ensure no stray token requirement blocks the read.
	t.Setenv("VAULT_TOKEN", "test-token")
}

// TestVaultProvider_GetDSN_ReturnsRawJSONPayload tests that the vaultProvider returns the raw JSON payload
func TestVaultProvider_GetDSN_ReturnsRawJSONPayload(t *testing.T) {
	srv := newVaultTestServer(t, map[string]any{
		"writer": "postgres://writer-dsn",
		"reader": "postgres://reader-dsn",
	})
	defer srv.Close()
	setVaultAddrEnv(t, srv.URL)

	ref, err := url.Parse("hashivault://secret/my-db-secret")
	if err != nil {
		t.Fatalf("parse ref: %v", err)
	}

	p := vaultProvider{}
	raw, err := p.getDSN(context.Background(), ref)
	if err != nil {
		t.Fatalf("getDSN returned error: %v", err)
	}

	var payload map[string]string
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		t.Fatalf("expected raw JSON payload, got %q (unmarshal err: %v)", raw, err)
	}
	if payload["writer"] != "postgres://writer-dsn" {
		t.Fatalf("writer mismatch: got %q", payload["writer"])
	}
	if payload["reader"] != "postgres://reader-dsn" {
		t.Fatalf("reader mismatch: got %q", payload["reader"])
	}
}

// TestVaultProvider_GetDSN_KVv1 tests that the vaultProvider correctly handles KV v1 secrets.
func TestVaultProvider_GetDSN_KVv1(t *testing.T) {
	srv := newVaultTestServerV1(t, map[string]any{
		"data_source_name": "postgres://v1-dsn",
	})
	defer srv.Close()
	setVaultAddrEnv(t, srv.URL)

	ref, err := url.Parse("hashivault://secret/my-db-secret?engine_version=1")
	if err != nil {
		t.Fatalf("parse ref: %v", err)
	}

	p := vaultProvider{}
	raw, err := p.getDSN(context.Background(), ref)
	if err != nil {
		t.Fatalf("getDSN returned error: %v", err)
	}

	var payload map[string]string
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		t.Fatalf("expected raw JSON payload, got %q (unmarshal err: %v)", raw, err)
	}
	if payload["data_source_name"] != "postgres://v1-dsn" {
		t.Fatalf("data_source_name mismatch: got %q", payload["data_source_name"])
	}
}

// TestVaultProvider_GetDSN_NonStringValue_Errors tests that the vaultProvider returns an error when a secret value is
// not a string.
func TestVaultProvider_GetDSN_NonStringValue_Errors(t *testing.T) {
	srv := newVaultTestServer(t, map[string]any{
		"writer": 12345, // non-string value should trigger an error
	})
	defer srv.Close()
	setVaultAddrEnv(t, srv.URL)

	ref, err := url.Parse("hashivault://secret/my-db-secret")
	if err != nil {
		t.Fatalf("parse ref: %v", err)
	}

	p := vaultProvider{}
	_, err = p.getDSN(context.Background(), ref)
	if err == nil {
		t.Fatal("expected error for non-string secret value, got nil")
	}
	if !strings.Contains(err.Error(), "is not a string") {
		t.Fatalf("unexpected error message: %v", err)
	}
}

// TestVaultProvider_GetDSN_SecretFetchError tests that the vaultProvider returns an error when the secret cannot be
// fetched.
func TestVaultProvider_GetDSN_SecretFetchError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()
	setVaultAddrEnv(t, srv.URL)

	ref, err := url.Parse("hashivault://secret/missing-secret")
	if err != nil {
		t.Fatalf("parse ref: %v", err)
	}

	p := vaultProvider{}
	_, err = p.getDSN(context.Background(), ref)
	if err == nil {
		t.Fatal("expected error for missing secret, got nil")
	}
	if !strings.Contains(err.Error(), "unable to read Vault secret") {
		t.Fatalf("unexpected error message: %v", err)
	}
}

// TestVaultProvider_HonorsVaultAddrEnv tests that the vaultProvider honors the VAULT_ADDR environment variable.
func TestVaultProvider_HonorsVaultAddrEnv(t *testing.T) {
	srv := newVaultTestServer(t, map[string]any{"data_source_name": "x"})
	defer srv.Close()
	setVaultAddrEnv(t, srv.URL)

	cfg := vault.DefaultConfig()
	if err := cfg.ReadEnvironment(); err != nil {
		t.Fatalf("ReadEnvironment: %v", err)
	}
	if cfg.Address != srv.URL {
		t.Fatalf("expected address %q, got %q", srv.URL, cfg.Address)
	}
}
