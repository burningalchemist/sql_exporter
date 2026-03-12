# SQL Exporter Examples

This directory contains example configurations for various use cases of the SQL Exporter.

## Directory Structure

### Database-specific Examples
- **`postgres/`** - PostgreSQL-specific collectors and queries
- **`mssql/`** - Microsoft SQL Server collectors
- **`azure-sql-mi/`** - Azure SQL Managed Instance with Grafana dashboards

### Helm Configuration Examples
- **`tls-only/`** - TLS encryption with certificates from Kubernetes secret
- **`auth-only/`** - Basic authentication with bcrypt password hashing
- **`dynamic-config-only/`** - Dynamic configuration from external secret (DSN)
- **`tls-auth-dynamic/`** - Combined TLS, authentication, and dynamic config

## Contributing

When adding new examples:
1. Create a new subdirectory
2. Include `values.yaml`, `README.md`, and any `secret-*.yaml` files
3. Update this main README with a link
4. Test the example in a real Kubernetes cluster
5. Document all prerequisites and customization options
