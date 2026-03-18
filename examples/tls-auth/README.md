# Example: TLS + Authentication

This example demonstrates the full security configuration
- TLS encryption for metrics endpoint
- Basic authentication for access control
- Static configuration with predefined collectors
- Shared secret for both TLS certificates and authentication password

## Use Case

- **Production deployment** requiring comprehensive security
- Database credentials defined at deploy time 
- Need both encryption (TLS) and authentication (basic auth)
- Collectors and targets are known and don't change at runtime

## Files

- **`values-example.yaml`** - Complete Helm values with TLS, auth, and static config
- **`secret-tls-auth.yaml`** - Shared secret example for TLS + password

## Prerequisites

### 1. Create shared secret with TLS + password

```bash
kubectl create secret generic sql-exporter-tls-auth \
  --from-file=tls.crt=path/to/cert.crt \
  --from-file=tls.key=path/to/cert.key \
  --from-literal=password='your-secure-password' \
  --namespace=your-namespace
```

For self-signed certificates:
```bash
openssl req -x509 -newkey rsa:4096 -keyout tls.key -out tls.crt -days 365 -nodes
kubectl create secret generic sql-exporter-tls-auth \
  --from-file=tls.crt=tls.crt \
  --from-file=tls.key=tls.key \
  --from-literal=password='your-secure-password' \
  --namespace=your-namespace
```

For detailed TLS certificate options, see `secret-tls-auth.yaml`.

## Deployment

```bash
helm install sql-exporter ../../helm -f values-example.yaml
```

## Key Features

- **HTTPS metrics endpoint** with TLS 1.3 encryption
- **Basic authentication** with bcrypt-hashed passwords
- **Static config** with predefined collectors and database target
- **Shared secret** consolidation (one secret for TLS + auth)
- **Single init container** for password hashing
- **tcpSocket health probes** (httpGet doesn't support auth)
- Production-ready with resource limits
- ServiceMonitor for Prometheus Operator

## Verification

```bash
# Check pod status
kubectl get pods -l app.kubernetes.io/name=sql-exporter

# Check pod is ready
kubectl describe pod <pod-name>

# Verify config
kubectl exec <pod-name> -- cat /etc/sql_exporter/sql_exporter.yml

# Test metrics endpoint (with TLS + auth)
kubectl port-forward svc/sql-exporter 9399:9399
curl -k -u prometheus:your-secure-password https://localhost:9399/metrics
```

## Technical Details

### Secret Consolidation

This example demonstrates **shared secret** usage:
- `sql-exporter-tls-auth` provides:
  - `tls.crt` - TLS certificate
  - `tls.key` - TLS private key
  - `password` - Plaintext password for basic auth

This single secret is:
1. Mounted at `/tls` for main container (TLS certs)
2. Mounted at `/secret-src` for init container (password)

### Init Container Flow

**Init Container** (`sql-exporter-init`):
1. Reads plaintext password from `sql-exporter-tls-auth` secret
2. Hashes password using bcrypt (cost: 12)
3. Reads TLS config template (base64-encoded)
4. Appends `basic_auth_users` section with hashed password
5. Writes `web-config.yml` to `/etc/web-config/` (emptyDir)

**Main Container**:
- Uses `/etc/sql_exporter/sql_exporter.yml` for SQL exporter config
- Uses `/etc/web-config/web-config.yml` for web server config (TLS + auth)
- Mounts TLS certs from `/tls`

### Health Probes

When basic auth is enabled, probes use `tcpSocket` instead of `httpGet`:
- **Why?** Kubernetes `httpGet` probes don't support authentication headers
- **Effect:** Opens TCP connection to port 9399 but doesn't complete TLS handshake
- **Logs:** You'll see harmless `TLS handshake error: EOF` messages - this is expected

## Customization

Edit `values-example.yaml` to:
- Add/modify collectors in `config.collectors`
- Change database target in `config.target.data_source_name`
- Adjust scrape intervals (`min_interval`)
- Change bcrypt cost (higher = more secure but slower)
- Configure resource limits
- Add ServiceMonitor selector labels for Prometheus discovery
- Change log level (`logLevel: info` or `debug`)

## Production Considerations

✅ **This example includes:**
- TLS 1.3 encryption with strong cipher suites
- Password hashing with bcrypt
- Static credential management via secrets
- Resource limits
- ServiceMonitor for Prometheus Operator
- Pod labels for organization

## Troubleshooting

**Pod stuck in Init:0/1:**
- Check if `sql-exporter-tls-auth` secret exists
- Verify it contains `tls.crt`, `tls.key`, and `password` keys
- Check init container logs: `kubectl logs <pod> -c sql-exporter-init`

**Metrics return 401 Unauthorized:**
- Verify username/password are correct
- Check `web-config.yml` was generated: `kubectl exec <pod> -- cat /etc/web-config/web-config.yml`
- Ensure password is correct in secret

**TLS handshake EOF errors in logs:**
- These are harmless and expected from `tcpSocket` health probes
- Can be reduced by setting `logLevel: info` or `warn`

**Pod fails to start - metrics collection error:**
- Verify database target DSN in `config.target.data_source_name`
- Ensure database is reachable from the pod
- Check pod logs: `kubectl logs <pod> -c sql-exporter`

## Security Notes

- Plaintext password is only in secret, never in logs
- Password is bcrypt-hashed before being written to config
- TLS certificates should be properly signed (use cert-manager for production)
- All communication to metrics endpoint is encrypted and authenticated

