# Kubernetes Secret DSN Resolution

SQL Exporter can read database connection strings (DSN) directly from Kubernetes secrets.

## Requirements

The service account running the pod must have permission to read secrets:
- `automountServiceAccountToken: true` - Automatically mounts the service account token
- RBAC Role with `secrets: get` permission

The provided Helm chart (`helm/templates/serviceaccount.yaml`) automatically creates the necessary Role and RoleBinding when `serviceAccount.create: true`.

## URL Format

```
k8ssecret://[namespace/]secret-name?key=field_name&[template=template_string]
```

**Parameters:**
- `namespace` (optional): Kubernetes namespace. If omitted, uses the pod's current namespace
- `secret-name`: Name of the Kubernetes secret (required)
- `key` (optional): Key within the secret to extract (defaults to `data_source_name`)
- `template` (optional): Template string using `DSN_VALUE` as placeholder for the secret value. If omitted, the secret value is used as-is

## Examples

### Full DSN Stored in Secret

Store a complete DSN in the secret:

```bash
kubectl create secret generic postgres-db \
  --from-literal=data_source_name='postgres://user:password@host:5432/mydb?sslmode=require'
```

Use it directly (no template needed):

```yaml
config:
  target:
    data_source_name: 'k8ssecret://postgres-db'
    collectors:
      - collector1
```

### Partial DSN with Template

Store only the credentials and connection details, build the full DSN with template:

```bash
kubectl create secret generic db-creds \
  --from-literal=APP_DB_CONNECTION='user:password@host:5432/database'
```

Build the full DSN with `postgres://` prefix and query parameters:

```yaml
config:
  target:
    data_source_name: 'k8ssecret://db-creds?key=APP_DB_CONNECTION&template=postgres://DSN_VALUE?application_name=sql-exporter&sslmode=require'
    collectors:
      - collector1
```

The `DSN_VALUE` placeholder will be replaced with the secret's value:
- Secret value: `user:password@host:5432/database`
- Result DSN: `postgres://user:password@host:5432/database?application_name=sql-exporter&sslmode=require`

### Cross-Namespace Secret

```yaml
config:
  target:
    data_source_name: 'k8ssecret://monitoring/db-secret'  # From 'monitoring' namespace
    collectors:
      - collector1
```

⚠️ **Note**: Accessing secrets from a different namespace is **not recommended** for production deployments. It requires additional RBAC ClusterRole with cross-namespace permissions that are **not provided** with this Helm chart. For cross-namespace access, you would need to manually create a ClusterRole with `secrets: get` permission across all namespaces. It's recommended to always store secrets in the same namespace as the SQL Exporter pod.

## Deployment

Use the provided values file for quick deployment:

```bash
helm install sql-exporter ./helm \
  -f deployment/values-override-static-config.yaml \
  -n your-namespace
```

This automatically:
1. Creates a service account with `automountServiceAccountToken: true`
2. Creates the necessary RBAC Role and RoleBinding for secret access
3. Configures the DSN to read from the Kubernetes secret
