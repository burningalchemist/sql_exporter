# Example: Basic Authentication Only

This example demonstrates how to deploy SQL Exporter with basic authentication for the metrics endpoint, without TLS encryption.

## Use Case

- You need access control via username/password
- TLS is handled at infrastructure level (e.g., service mesh, ingress with TLS termination)
- Want to restrict who can access metrics

## Files

- **`values-example.yaml`** - Helm values file configuring basic auth
- **`secret-auth.yaml`** - Complete guide for creating auth password secret (multiple methods)

## Prerequisites

Create a Kubernetes secret with plaintext password:

```bash
kubectl create secret generic sql-exporter-auth \
  --from-literal=password='your-secure-password' \
  --namespace=your-namespace
```

For more options (External Secrets, Sealed Secrets), see `secret-auth.yaml`.

## Deployment

```bash
helm install sql-exporter ../../helm -f values-example.yaml
```

## Key Features

- **Basic authentication** with username/password
- Password automatically hashed with bcrypt at pod startup (cost: 12)
- HTTP metrics endpoint (no TLS)
- Health probes use `tcpSocket` (httpGet doesn't support auth headers)
- Init container reads plaintext password and generates bcrypt hash

## Verification

```bash
# Check pod status
kubectl get pods -l app.kubernetes.io/name=sql-exporter

# Test metrics endpoint (with auth)
kubectl port-forward svc/sql-exporter 9399:9399
curl -u prometheus:your-secure-password http://localhost:9399/metrics
```

## Important Notes

**Prometheus ServiceMonitor with Basic Auth:**  
ServiceMonitor supports basic auth credentials only when referenced from a Kubernetes secret. You'll need to:
1. Create a secret with username and password for Prometheus to use
2. Configure the ServiceMonitor to reference this secret via `basicAuth` field
3. This is separate from the password secret used by sql-exporter itself

For production, consider using TLS + auth combination (`tls-auth-dynamic` example) for better security.

**Security Considerations:**
- Password is transmitted in plaintext (no TLS) - use only in trusted networks
- For production, **strongly recommend** using TLS + auth combination
- See `../tls-auth-dynamic/` example for complete security

## Customization

Edit `values-example.yaml` to:
- Change username (default: `prometheus`)
- Change database connection string
- Adjust bcrypt cost (higher = more secure but slower)
- Add/modify collectors
- Configure resource limits

## How It Works

1. Init container (`httpd:alpine`) runs at pod startup
2. Reads plaintext password from secret
3. Hashes password using `htpasswd` with bcrypt
4. Writes `web-config.yml` with hashed password to emptyDir
5. Main container mounts the generated web-config and enforces auth

## Logs

You may see harmless TLS handshake EOF errors from tcpSocket probes - this is expected behavior.

