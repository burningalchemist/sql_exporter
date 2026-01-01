# Example: TLS + Authentication + Dynamic Config (Complete Security)

This example demonstrates the full security configuration, combining all advanced features:
- TLS encryption for metrics endpoint
- Basic authentication for access control
- Dynamic configuration from database secret
- Shared secret for both TLS certificates and auth password

## Use Case

- **Production deployment** requiring comprehensive security
- Database credentials managed externally
- Need both encryption (TLS) and authentication (basic auth)
- Want to demonstrate secret consolidation best practices

## Files

- **`values-example.yaml`** - Complete Helm values with all features enabled
- **`secret-tls-auth.yaml`** - Shared secret for TLS certs + password
- **`secret-database.yaml`** - Database DSN secret

## Prerequisites

### 1. Create shared secret with TLS + password

```bash
kubectl create secret generic sql-exporter-tls-auth \
  --from-file=tls.crt=path/to/cert.crt \
  --from-file=tls.key=path/to/cert.key \
  --from-literal=password='your-secure-password' \
  --namespace=your-namespace
```

For detailed TLS certificate options, see `secret-tls-auth.yaml`.

### 2. Create database credentials secret

```bash
kubectl create secret generic database-credentials \
  --from-literal=dsn='username:password@hostname:5432/database?sslmode=disable' \
  --namespace=your-namespace
```

For different database types, see `secret-database.yaml`.

## Deployment

```bash
helm install sql-exporter ../../helm -f values-example.yaml
```

## Key Features

- **HTTPS metrics endpoint** with TLS 1.3 encryption
- **Basic authentication** with bcrypt-hashed passwords
- **Dynamic config** generated from database DSN at runtime
- **Shared secret** consolidation (one secret for TLS + auth)
- **Two init containers**:
  1. `sql-exporter-config-from-secret`: Generates `sql_exporter.yml`
  2. `webconfig-bcrypt`: Hashes password and generates `web-config.yml`
- **tcpSocket health probes** (httpGet doesn't support auth)
- Production-ready with resource limits

## Verification

```bash
# Check pod status
kubectl get pods -l app.kubernetes.io/name=sql-exporter

# Check init containers completed
kubectl describe pod <pod-name>

# Verify generated configs
kubectl exec <pod-name> -- cat /etc/sql_exporter/sql_exporter.yml
kubectl exec <pod-name> -- cat /etc/web-config/web-config.yml

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

**Init Container 1** (`sql-exporter-config-from-secret`):
1. Reads DSN from `database-credentials` secret
2. Reads template from Helm values (base64-encoded)
3. Substitutes `__TYPE__` and `__DSN__` placeholders
4. Writes `sql_exporter.yml` to `/etc/sql_exporter/` (emptyDir)

**Init Container 2** (`webconfig-bcrypt`):
1. Reads plaintext password from `sql-exporter-tls-auth` secret
2. Hashes password using `htpasswd` with bcrypt (cost: 12)
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
- Add/modify collectors in `dynamicConfig.template`
- Adjust scrape intervals (`min_interval`)
- Change bcrypt cost (higher = more secure but slower)
- Configure resource limits
- Add ServiceMonitor selector labels for Prometheus discovery
- Change log level (`logLevel: info` or `debug`)

## Production Considerations

✅ **This example includes:**
- TLS 1.3 encryption with strong cipher suites
- Password hashing with bcrypt
- Dynamic credential management
- Resource limits
- ServiceMonitor for Prometheus Operator
- Pod labels for organization

## Troubleshooting

**Pod stuck in Init:0/2:**
- Check if `database-credentials` secret exists
- Verify secret is in same namespace
- Check init container logs: `kubectl logs <pod> -c sql-exporter-config-from-secret`

**Pod stuck in Init:1/2:**
- Check if `sql-exporter-tls-auth` secret exists
- Verify it contains `tls.crt`, `tls.key`, and `password` keys
- Check init container logs: `kubectl logs <pod> -c webconfig-bcrypt`

**Metrics return 401 Unauthorized:**
- Verify username/password are correct
- Check `web-config.yml` was generated: `kubectl exec <pod> -- cat /etc/web-config/web-config.yml`
- Ensure password is base64-encoded in secret

**TLS handshake EOF errors in logs:**
- These are harmless and expected from `tcpSocket` health probes
- Can be reduced by setting `logLevel: info` or `warn`

## Security Notes

- Plaintext password is only in secret, never in logs
- Password is bcrypt-hashed before being written to config
- TLS certificates should be properly signed (use cert-manager)

