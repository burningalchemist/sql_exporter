# Example: TLS Encryption Only

This example demonstrates how to deploy SQL Exporter with TLS encryption for the metrics endpoint, without basic authentication.

## Use Case

- You want encrypted metrics transport (HTTPS)
- Authentication is handled at the network/infrastructure level (e.g., network policies, service mesh, ingress controller)
- Simplest security configuration

## Files

- **`values-example.yaml`** - Helm values file configuring TLS-only
- **`secret-tls.yaml`** - Complete guide for creating TLS secrets (multiple methods)

## Prerequisites

Create a Kubernetes secret with TLS certificates:

```bash
kubectl create secret tls sql-exporter-tls \
  --cert=path/to/cert.crt \
  --key=path/to/cert.key \
  --namespace=your-namespace
```

For more options (cert-manager, self-signed, YAML manifest, CA certificates), see `secret-tls.yaml`.

## Deployment

```bash
helm install sql-exporter ../../helm -f values-example.yaml
```

## Key Features

- **HTTPS metrics endpoint** with TLS 1.3
- Health probes automatically use HTTPS with certificate validation skip
- Static configuration with inline collectors
- Compatible with Prometheus ServiceMonitor (native HTTPS support)

## Verification

```bash
# Check pod status
kubectl get pods -l app.kubernetes.io/name=sql-exporter

# Test metrics endpoint (with TLS)
kubectl port-forward svc/sql-exporter 9399:9399
curl -k https://localhost:9399/metrics
```

## TLS Customization

Edit `values-example.yaml` to customize TLS settings:
- **Change TLS secret name**: `webConfig.tls.secretName`
- **Use different key names in secret**: `webConfig.tls.certKey` and `webConfig.tls.keyKey`
- **Change projected filenames**: `webConfig.tls.certFile` and `webConfig.tls.keyFile`
- **Customize TLS config template**: Override `webConfig.template` to specify different:
  - `min_version`: `TLS10`, `TLS11`, `TLS12`, or `TLS13` (default)
  - `cipher_suites`: Custom cipher suite list
  - `prefer_server_cipher_suites`: Server vs client cipher preference

## Notes

- Certificates must be in PEM format
- The secret must be in the same namespace as the deployment
- For production, use proper CA-signed certificates or cert-manager
- Kubernetes httpGet probes natively support HTTPS and skip certificate validation

