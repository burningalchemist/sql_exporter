# Example: Dynamic Configuration Only

This example demonstrates how to deploy SQL Exporter with dynamically generated configuration from a database DSN stored in a Kubernetes secret.

## Use Case

- Database credentials are managed externally (operator, service broker, secrets manager)
- Don't want to hardcode connection strings in Helm values
- Need to regenerate config when database credentials change (pod restart)
- Simplest deployment without TLS or authentication

## Files

- **`values-example.yaml`** - Helm values file with dynamic config template
- **`secret-database.yaml`** - Complete guide for creating database DSN secret (multiple methods)

## Prerequisites

Create a Kubernetes secret with database connection string (DSN):

```bash
# PostgreSQL
kubectl create secret generic database-credentials \
  --from-literal=dsn='username:password@hostname:5432/database?sslmode=disable' \
  --namespace=your-namespace

# MySQL
kubectl create secret generic database-credentials \
  --from-literal=dsn='username:password@tcp(hostname:3306)/database' \
  --namespace=your-namespace
```

For more database types and options (External Secrets, Service Binding), see `secret-database.yaml`.

## Deployment

```bash
helm install sql-exporter ../../helm -f values-example.yaml
```

## Key Features

- **Config generated at runtime** from database DSN in secret
- Init container substitutes `__TYPE__` and `__DSN__` placeholders
- Collectors defined in Helm values template
- HTTP metrics endpoint (no security)
- Supports multiple scrape intervals via `min_interval`

## Verification

```bash
# Check pod status and init container
kubectl get pods -l app.kubernetes.io/name=sql-exporter
kubectl logs <pod-name> -c sql-exporter-config-from-secret

# Verify generated config
kubectl exec <pod-name> -- cat /etc/sql_exporter/sql_exporter.yml

# Test metrics endpoint
kubectl port-forward svc/sql-exporter 9399:9399
curl http://localhost:9399/metrics
```

## DSN Format by Database

The DSN should NOT include the driver prefix (e.g., `postgres://`). The init container adds it based on `dynamicConfig.type`.

**PostgreSQL:**
```
username:password@hostname:5432/database?sslmode=disable
```

**MySQL:**
```
username:password@tcp(hostname:3306)/database
```

**MSSQL:**
```
username:password@hostname:1433?database=dbname
```

For more formats, see `secret-database.yaml`.

## Customization

Edit `values-example.yaml` to:
- Change database type (`dynamicConfig.type`)
- Add/modify collectors in `dynamicConfig.template`
- Adjust scrape intervals
- Configure resource limits
- Add ServiceMonitor labels

## How It Works

1. Init container (`alpine:3.19`) runs at pod startup
2. Reads DSN from secret specified in `dynamicConfig.secretName`
3. Base64-decodes the template from `dynamicConfig.template`
4. Substitutes `__TYPE__` with `dynamicConfig.type` (e.g., `postgres`)
5. Substitutes `__DSN__` with the actual DSN from secret
6. Writes final `sql_exporter.yml` to emptyDir at `/etc/sql_exporter/`
7. Main container starts and uses the generated config

## When to Use Dynamic Config

✅ **Use when:**
- Database credentials change frequently
- Using operators/service brokers that manage credentials
- Want to avoid hardcoding sensitive data in Helm values
- Need to integrate with external secrets management

❌ **Don't use when:**
- Static config is sufficient
- You manage credentials directly in Helm values
- Config changes require application logic changes (not just credentials)

## Security Notes

- This example has NO encryption or authentication
- For production, combine with TLS and/or basic auth
- See `../tls-auth-dynamic/` for complete security example
- Consider using External Secrets Operator or similar for secret management

