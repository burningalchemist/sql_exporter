package config

import (
	"context"
	"encoding/json"
	"net/url"
	"strings"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

// newTestK8sProvider builds a k8sSecretProvider via a fake clientset, bypassing getK8sProvider in-cluster setup
func newTestK8sProvider(namespace string, objects ...*corev1.Secret) k8sSecretProvider {
	// k8sSecretProvider.getDSN calls getK8sProvider() internally; to keep this test hermetic we exercise the
	// provider's logic via a lightweight wrapper. 
	return k8sSecretProvider{}
}

// newFakeClientsetWithSecret creates a fake Kubernetes clientset with a single Secret object for testing.
func newFakeClientsetWithSecret(namespace, name string, data map[string][]byte, stringData map[string]string) *fake.Clientset {
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Data:       data,
		StringData: stringData,
	}
	return fake.NewSimpleClientset(secret)
}

// TestK8sSecretProvider_GetDSN_ReturnsRawJSONPayload tests that getDSN returns the raw JSON payload of the secret.
func TestK8sSecretProvider_GetDSN_ReturnsRawJSONPayload(t *testing.T) {
	clientset := newFakeClientsetWithSecret("default", "my-db-secret",
		map[string][]byte{
			"writer": []byte("postgres://writer-dsn"),
		},
		map[string]string{
			"reader": "postgres://reader-dsn",
		},
	)

	k8sProviderInstance = &k8sSecretProvider{
		clientset: clientset,
		namespace: "default",
	}
	t.Cleanup(func() { k8sProviderInstance = nil })

	ref, err := url.Parse("k8ssecret://default/my-db-secret")
	if err != nil {
		t.Fatalf("parse ref: %v", err)
	}

	p := k8sSecretProvider{}
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

// TestK8sSecretProvider_GetDSN_CurrentNamespaceShorthand tests that getDSN can infer the namespace when using the
// shorthand URL format.
func TestK8sSecretProvider_GetDSN_CurrentNamespaceShorthand(t *testing.T) {
	clientset := newFakeClientsetWithSecret("current-ns", "db-creds",
		map[string][]byte{"data_source_name": []byte("postgres://short-dsn")},
		nil,
	)

	k8sProviderInstance = &k8sSecretProvider{
		clientset: clientset,
		namespace: "current-ns",
	}
	t.Cleanup(func() { k8sProviderInstance = nil })

	// Single-segment URL: k8ssecret://db-creds (namespace inferred).
	ref, err := url.Parse("k8ssecret://db-creds")
	if err != nil {
		t.Fatalf("parse ref: %v", err)
	}

	p := k8sSecretProvider{}
	raw, err := p.getDSN(context.Background(), ref)
	if err != nil {
		t.Fatalf("getDSN returned error: %v", err)
	}

	var payload map[string]string
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		t.Fatalf("expected raw JSON payload, got %q (unmarshal err: %v)", raw, err)
	}
	if payload["data_source_name"] != "postgres://short-dsn" {
		t.Fatalf("data_source_name mismatch: got %q", payload["data_source_name"])
	}
}

// TestK8sSecretProvider_GetDSN_SecretNotFound tests that getDSN returns an error when the secret is not found.
func TestK8sSecretProvider_GetDSN_SecretNotFound(t *testing.T) {
	clientset := fake.NewSimpleClientset() // no secrets

	k8sProviderInstance = &k8sSecretProvider{
		clientset: clientset,
		namespace: "default",
	}
	t.Cleanup(func() { k8sProviderInstance = nil })

	ref, err := url.Parse("k8ssecret://default/missing-secret")
	if err != nil {
		t.Fatalf("parse ref: %v", err)
	}

	p := k8sSecretProvider{}
	_, err = p.getDSN(context.Background(), ref)
	if err == nil {
		t.Fatal("expected error for missing secret, got nil")
	}
	if !strings.Contains(err.Error(), "unable to fetch secret") {
		t.Fatalf("unexpected error message: %v", err)
	}
}

// TestK8sSecretProvider_GetDSN_InvalidURL tests that getDSN returns an error for an invalid k8ssecret URL.
func TestK8sSecretProvider_GetDSN_InvalidURL(t *testing.T) {
	k8sProviderInstance = &k8sSecretProvider{
		clientset: fake.NewSimpleClientset(),
		namespace: "default",
	}
	t.Cleanup(func() { k8sProviderInstance = nil })

	ref, err := url.Parse("k8ssecret://")
	if err != nil {
		t.Fatalf("parse ref: %v", err)
	}

	p := k8sSecretProvider{}
	_, err = p.getDSN(context.Background(), ref)
	if err == nil {
		t.Fatal("expected error for invalid k8ssecret URL, got nil")
	}
	if !strings.Contains(err.Error(), "invalid k8ssecret URL format") {
		t.Fatalf("unexpected error message: %v", err)
	}
}

// TestSecretResolver_K8sSecret_MultipleJobsDifferentKeys tests that the secretResolver correctly resolves different
// keys from the same k8ssecret across multiple jobs without caching issues.
func TestSecretResolver_K8sSecret_MultipleJobsDifferentKeys(t *testing.T) {
	clientset := newFakeClientsetWithSecret("default", "my-db-secret",
		map[string][]byte{
			"writer": []byte("postgres://writer-dsn"),
			"reader": []byte("postgres://reader-dsn"),
		},
		nil,
	)

	k8sProviderInstance = &k8sSecretProvider{
		clientset: clientset,
		namespace: "default",
	}
	t.Cleanup(func() { k8sProviderInstance = nil })

	origProviders := secretProviders
	defer func() { secretProviders = origProviders }()
	secretProviders = map[string]secretProvider{
		"k8ssecret": k8sSecretProvider{},
	}

	r := &secretResolver{}
	ctx := context.Background()

	// Simulates job1's static_config DSN.
	gotWriter, err := r.resolve(ctx, "k8ssecret://default/my-db-secret?key=writer")
	if err != nil {
		t.Fatalf("resolve writer (job1) returned error: %v", err)
	}
	if gotWriter != "postgres://writer-dsn" {
		t.Fatalf("job1 (writer) DSN mismatch: got %q want %q", gotWriter, "postgres://writer-dsn")
	}

	// Simulates job2's static_config DSN, same secret, different key. Before the fix, this would incorrectly return
	// job1's writer DSN.
	gotReader, err := r.resolve(ctx, "k8ssecret://default/my-db-secret?key=reader")
	if err != nil {
		t.Fatalf("resolve reader (job2) returned error: %v", err)
	}
	if gotReader != "postgres://reader-dsn" {
		t.Fatalf("job2 (reader) DSN mismatch: got %q want %q (bug: got job1's cached DSN)", gotReader, "postgres://reader-dsn")
	}

	// A third job re-referencing job1's key should still work from cache.
	gotWriterAgain, err := r.resolve(ctx, "k8ssecret://default/my-db-secret?key=writer")
	if err != nil {
		t.Fatalf("resolve writer (job3) returned error: %v", err)
	}
	if gotWriterAgain != "postgres://writer-dsn" {
		t.Fatalf("job3 (writer, cached) DSN mismatch: got %q want %q", gotWriterAgain, "postgres://writer-dsn")
	}

	// Confirm only one underlying k8s API call was made (shared fetch), not one per job/key.
	getActions := 0
	for _, action := range clientset.Actions() {
		if action.GetVerb() == "get" && action.GetResource().Resource == "secrets" {
			getActions++
		}
	}
	if getActions != 1 {
		t.Fatalf("expected exactly 1 k8s secret fetch (shared across jobs), got %d", getActions)
	}
}
