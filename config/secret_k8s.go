package config

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

type k8sSecretProvider struct {
	clientset kubernetes.Interface
	namespace string // cached current namespace
}

var k8sProviderInstance *k8sSecretProvider

// getCurrentNamespace retrieves the current pod's namespace from the downward API.
func getCurrentNamespace() (string, error) {
	nsBytes, err := os.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/namespace")
	if err != nil {
		return "", fmt.Errorf("unable to read current namespace: %w", err)
	}
	return string(nsBytes), nil
}

// getK8sProvider returns a singleton instance of the k8s secret provider, lazily initializing the client.
func getK8sProvider() (*k8sSecretProvider, error) {
	if k8sProviderInstance != nil {
		return k8sProviderInstance, nil
	}

	// Use in-cluster configuration
	config, err := rest.InClusterConfig()
	if err != nil {
		return nil, fmt.Errorf("unable to load in-cluster Kubernetes config: %w", err)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("unable to create Kubernetes client: %w", err)
	}

	// Get current namespace
	namespace, err := getCurrentNamespace()
	if err != nil {
		return nil, err
	}

	k8sProviderInstance = &k8sSecretProvider{
		clientset: clientset,
		namespace: namespace,
	}
	return k8sProviderInstance, nil
}

// getDSN fetches a plain string value from a Kubernetes secret.
// URL format: k8ssecret://[namespace/]secret-name?key=field&template=dsn_template
//
// Examples (with explicit namespace):
//   - k8ssecret://default/my-db-secret
//   - k8ssecret://monitoring/db-creds?key=password&template=postgres://user:DSN_VALUE@host:5432/db
//
// Examples (current namespace, auto-detected):
//   - k8ssecret://my-db-secret
//   - k8ssecret://db-creds?key=password
//
// Parameters:
//   - namespace: Kubernetes namespace (optional, defaults to pod's current namespace)
//   - secret-name: Name of the Kubernetes secret (required)
//   - key: The key within the secret to extract (optional, defaults to "data_source_name")
//   - template: Template string for building the DSN (optional, uses DSN_VALUE as placeholder for secret value)
//
// The secret value is returned as-is (plain string). If template is provided, it replaces
// all occurrences of DSN_VALUE with the actual secret value.
func (p k8sSecretProvider) getDSN(ctx context.Context, ref *url.URL) (string, error) {
	provider, err := getK8sProvider()
	if err != nil {
		return "", err
	}

	namespace := ref.Host
	secretName := ref.Path

	// Remove leading slash from secret name if present
	if secretName != "" && secretName[0] == '/' {
		secretName = secretName[1:]
	}

	// If namespace is empty or looks like a secret name (no slashes), treat Host as secret name in current namespace
	if namespace == "" || (ref.Path == "" && namespace != "") {
		// Single-part URL: k8ssecret://secret-name
		secretName = namespace
		namespace = provider.namespace
	}

	if secretName == "" {
		return "", fmt.Errorf("invalid k8ssecret URL format: expected k8ssecret://[namespace/]secret-name, got %s", ref.String())
	}

	// Extract the key from the secret data
	key := ref.Query().Get("key")
	if key == "" {
		key = "data_source_name"
	}

	// Fetch the secret from Kubernetes API
	secret, err := provider.clientset.CoreV1().Secrets(namespace).Get(ctx, secretName, metav1.GetOptions{})
	if err != nil {
		return "", fmt.Errorf("unable to fetch secret %q from namespace %q: %w", secretName, namespace, err)
	}

	// Extract the key from secret data - check both Data (binary) and StringData (string)
	var secretValue string

	// Check in Data field first (for binary/encoded data)
	if data, ok := secret.Data[key]; ok {
		secretValue = string(data)
	} else if stringData, ok := secret.StringData[key]; ok {
		// Check in StringData field (for direct string values)
		secretValue = stringData
	} else {
		return "", fmt.Errorf("key %q not found in Kubernetes secret %s/%s", key, namespace, secretName)
	}

	// Apply template if provided
	templateStr := ref.Query().Get("template")
	if templateStr != "" {
		// Simple string replacement - replace all occurrences of DSN_VALUE with the secret value
		result := strings.ReplaceAll(templateStr, "DSN_VALUE", secretValue)
		return result, nil
	}

	return secretValue, nil
}
