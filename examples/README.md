# SQL Exporter Examples

This directory contains example configurations for various use cases of the SQL Exporter.

## Directory Structure

### Database-specific Examples
- **`postgres/`** - PostgreSQL-specific collectors and queries
- **`mssql/`** - Microsoft SQL Server collectors
- **`azure-sql-mi/`** - Azure SQL Managed Instance with Grafana dashboards

### Helm Chart Security Examples

Each Helm example is in its own subdirectory with complete documentation and secret creation guides:

#### 1. [`tls-only/`](tls-only/) - TLS Encryption Only

Secures the `/metrics` endpoint with TLS encryption without authentication.

- **Use case**: Encrypted metrics transport, authentication handled elsewhere
- **Files**: `values-example.yaml`, `secret-tls.yaml`, `README.md`
- **Deploy**: `helm install sql-exporter ../../helm -f tls-only/values-example.yaml`

**Prerequisites**:
```bash
kubectl create secret tls sql-exporter-tls \
  --cert=path/to/cert.crt \
  --key=path/to/cert.key
```

---

#### 2. [`auth-only/`](auth-only/) - Basic Authentication Only

Protects the `/metrics` endpoint with username/password authentication.

- **Use case**: Access control, TLS handled at infrastructure level
- **Files**: `values-example.yaml`, `secret-auth.yaml`, `README.md`
- **Deploy**: `helm install sql-exporter ../../helm -f auth-only/values-example.yaml`

**Prerequisites**:
```bash
kubectl create secret generic sql-exporter-auth \
  --from-literal=password='your-secure-password'
```

**Note**: Prometheus ServiceMonitor doesn't natively support basic auth. Consider using TLS+Auth combination.

---

#### 3. [`dynamic-config-only/`](dynamic-config-only/) - Dynamic Configuration

Generates `sql_exporter.yml` at runtime from a database DSN stored in a Kubernetes secret.

- **Use case**: Database credentials managed externally, avoid hardcoding connection strings
- **Files**: `values-example.yaml`, `secret-database.yaml`, `README.md`
- **Deploy**: `helm install sql-exporter ../../helm -f dynamic-config-only/values-example.yaml`

**Prerequisites**:
```bash
kubectl create secret generic database-credentials \
  --from-literal=dsn='username:password@hostname:5432/database?sslmode=disable'
```

---

#### 4. [`tls-auth-dynamic/`](tls-auth-dynamic/) - Complete Security Configuration ⭐

Combines all security features for production deployment.

- **Use case**: Production deployment requiring comprehensive security
- **Features**:
  - ✅ HTTPS metrics endpoint with TLS 1.3
  - ✅ Basic authentication with bcrypt-hashed passwords
  - ✅ Dynamic config from database secret
  - ✅ Shared secret for both TLS certificates and auth password
  - ✅ Resource limits and production-ready configuration
- **Files**: `values-example.yaml`, `secret-tls-auth.yaml`, `secret-database.yaml`, `README.md`
- **Deploy**: `helm install sql-exporter ../../helm -f tls-auth-dynamic/values-example.yaml`

**Prerequisites**:
```bash
# Shared secret with TLS + password
kubectl create secret generic sql-exporter-tls-auth \
  --from-file=tls.crt=path/to/cert.crt \
  --from-file=tls.key=path/to/cert.key \
  --from-literal=password='your-secure-password'

# Database credentials
kubectl create secret generic database-credentials \
  --from-literal=dsn='username:password@hostname:5432/database?sslmode=disable'
```

---

## Quick Start

1. **Choose an example** based on your security requirements
2. **Navigate to the subdirectory** and read its README.md
3. **Create required secrets** (see prerequisites in each README)
4. **Customize values-example.yaml** for your environment
5. **Deploy** using Helm

## Comparison Table

| Example | TLS | Auth | Dynamic Config | Complexity | Production Ready |
|---------|-----|------|----------------|------------|------------------|
| `tls-only/` | ✅ | ❌ | ❌ | Low | ⚠️ (needs auth) |
| `auth-only/` | ❌ | ✅ | ❌ | Low | ⚠️ (needs TLS) |
| `dynamic-config-only/` | ❌ | ❌ | ✅ | Medium | ❌ (no security) |
| `tls-auth-dynamic/` | ✅ | ✅ | ✅ | High | ✅ |

## Common Tasks

### Testing Deployment
```bash
# Check pod status
kubectl get pods -l app.kubernetes.io/name=sql-exporter

# View logs
kubectl logs -l app.kubernetes.io/name=sql-exporter --tail=50

# Port forward and test
kubectl port-forward svc/sql-exporter 9399:9399
```

### Test Metrics Endpoint
```bash
# HTTP (no security)
curl http://localhost:9399/metrics

# HTTPS (TLS only)
curl -k https://localhost:9399/metrics

# HTTPS with basic auth
curl -k -u prometheus:password https://localhost:9399/metrics
```

### Verify Secrets
```bash
# List secrets
kubectl get secrets

# Describe secret
kubectl describe secret sql-exporter-tls

# View certificate details
kubectl get secret sql-exporter-tls -o jsonpath='{.data.tls\.crt}' | \
  base64 -d | openssl x509 -text -noout
```

## Secret Creation Methods

Each example directory contains detailed secret creation guides (`secret-*.yaml`) with multiple methods:

1. **kubectl create secret** (simplest, recommended for quick start)
2. **kubectl apply -f** (YAML manifest with base64-encoded values)
3. **cert-manager** (for TLS, automatic certificate management)
4. **External Secrets Operator** (production, integrates with Vault, AWS Secrets Manager, etc.)
5. **Sealed Secrets** (for GitOps workflows)

## Customization Guide

All examples can be customized by editing `values-example.yaml`:

- **Database connection**: Change `data_source_name` or DSN in secret
- **Collectors**: Add/modify under `collectors:` or `dynamicConfig.template`
- **Scrape intervals**: Set `min_interval` per collector
- **Resource limits**: Adjust `resources.limits` and `resources.requests`
- **Log level**: Change `logLevel` (`debug`, `info`, `warn`, `error`)
- **Probes**: Fine-tune timing or provide full custom probe configuration
- **ServiceMonitor**: Add `selector` labels for Prometheus discovery

## Production Recommendations

For production deployments, use [`tls-auth-dynamic/`](tls-auth-dynamic/) as the base and additionally:

1. ✅ **TLS**: Always use TLS for encrypted transport
2. ✅ **Authentication**: Enable basic auth for access control
3. ✅ **Dynamic config**: Use for externally managed credentials
4. ✅ **cert-manager**: Automatic certificate generation and renewal
5. ✅ **External Secrets**: Integrate with enterprise secret managers
6. ✅ **Resource limits**: Prevent resource exhaustion
7. ✅ **Network policies**: Restrict pod-to-pod communication
8. ✅ **ServiceMonitor**: Enable for Prometheus Operator
9. ✅ **Query review**: Test collector queries for performance impact
10. ✅ **Monitoring**: Set up alerts for exporter health and metrics

## Troubleshooting

### Pod Won't Start
- Check secrets exist in same namespace
- Verify secret key names match configuration
- Review init container logs: `kubectl logs <pod> -c <init-container-name>`

### Metrics Return Errors
- 401 Unauthorized: Check auth credentials
- Connection refused: Verify service and port configuration
- TLS errors: Check certificate validity and format

### Common Errors
- **"secret not found"**: Create required secrets first
- **"Init:Error"**: Init container failed, check logs
- **"ImagePullBackOff"**: Check image repository and pull secrets
- **"TLS handshake EOF"**: Harmless when using tcpSocket probes with TLS

## Additional Resources

- [SQL Exporter Documentation](https://github.com/burningalchemist/sql_exporter)
- [Helm Chart Values Reference](../helm/values.yaml)
- [Prometheus Exporter Toolkit](https://github.com/prometheus/exporter-toolkit) (web config spec)
- [Helm Documentation](https://helm.sh/docs/)
- [Kubernetes Secrets](https://kubernetes.io/docs/concepts/configuration/secret/)

## Contributing

When adding new examples:
1. Create a new subdirectory
2. Include `values.yaml`, `README.md`, and any `secret-*.yaml` files
3. Update this main README with a link
4. Test the example in a real Kubernetes cluster
5. Document all prerequisites and customization options
