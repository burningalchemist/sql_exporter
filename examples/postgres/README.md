# PostgreSQL Collector Examples

This directory contains example collector configurations for PostgreSQL databases.

## Overview

These collectors demonstrate common PostgreSQL monitoring patterns using `sql_exporter`. They can be used as-is or customized for your specific needs.

## Files

### `postgres_database.yml`
Database-level metrics including:
- Database size and growth
- Connection counts by database
- Transaction rates (commits, rollbacks)
- Tuple operations (inserts, updates, deletes)
- Cache hit ratios

### `postgres_server.yml`
Server-level metrics including:
- Connection states (active, idle, waiting)
- Background writer statistics
- Checkpointer activity
- WAL generation rates
- Replication lag (for replicas)

### `sql_exporter.yml`
Complete configuration example combining both collectors with connection settings.

## Usage with Helm Chart

### Method 1: Using `collectorFiles` (Recommended)

```yaml
collectorFiles:
  postgres_database.yml: |
    # Contents of postgres_database.yml
  postgres_server.yml: |
    # Contents of postgres_server.yml

config:
  target:
    data_source_name: "postgres://username:password@hostname:5432/postgres?sslmode=disable"
  collector_files:
    - "*.collector.yml"
```

### Method 2: Using Static Config

```yaml
config:
  target:
    data_source_name: "postgres://username:password@hostname:5432/postgres?sslmode=disable"
    collectors:
      - pg_database
      - pg_stat_activity
  collectors:
    # Inline collector definitions from postgres_database.yml and postgres_server.yml
```

### Method 3: Using Dynamic Config

```yaml
dynamicConfig:
  enabled: true
  type: postgres
  secretName: postgres-credentials
  secretKey: dsn
  template: |
    target:
      data_source_name: __TYPE__://__DSN__
      collectors:
        - pg_database
        - pg_stat_activity
    # Include collector definitions from postgres_*.yml files
```

## Connection String Format

```
postgres://username:password@hostname:5432/database?sslmode=disable
```

### SSL Modes
- `disable` - No SSL
- `require` - SSL required (no certificate validation)
- `verify-ca` - Validate CA certificate
- `verify-full` - Full certificate validation

## Common Customizations

### Filter Specific Databases
Modify queries to exclude system databases:

```sql
WHERE datname NOT IN ('postgres', 'template0', 'template1')
```

### Adjust Collection Intervals
Set different `min_interval` per collector:

```yaml
collectors:
  - collector_name: pg_database
    min_interval: 60s  # Collect every minute
    metrics:
      # ... metric definitions
```

### Add Custom Metrics
Extend collectors with your own queries:

```yaml
collectors:
  - collector_name: pg_custom
    metrics:
      - metric_name: custom_table_size
        type: gauge
        help: 'Custom table sizes'
        values: [size_bytes]
        query: |
          SELECT pg_total_relation_size('my_table') as size_bytes;
```

## Grafana Dashboards

These collectors are compatible with popular PostgreSQL Grafana dashboards:
- [PostgreSQL Database Dashboard](https://grafana.com/grafana/dashboards/9628)
- [PostgreSQL Exporter Quickstart](https://grafana.com/grafana/dashboards/12485)

## Permissions Required

Grant the monitoring user necessary permissions:

```sql
-- Create monitoring user
CREATE USER sql_exporter WITH PASSWORD 'secure_password';

-- Grant connection
GRANT CONNECT ON DATABASE postgres TO sql_exporter;

-- Grant read access to statistics views
GRANT pg_monitor TO sql_exporter;  -- PostgreSQL 10+

-- For PostgreSQL 9.6 and earlier:
GRANT SELECT ON pg_stat_database TO sql_exporter;
GRANT SELECT ON pg_stat_activity TO sql_exporter;
GRANT SELECT ON pg_stat_replication TO sql_exporter;
```

## Testing

Test the collector configuration:

```bash
# Test connection
psql -h hostname -U sql_exporter -d postgres -c "SELECT version();"

# Test queries manually
psql -h hostname -U sql_exporter -d postgres -f postgres_database.yml
```

## Troubleshooting

### Connection Issues
- Verify `pg_hba.conf` allows connections from the SQL Exporter pod
- Check firewall rules allow PostgreSQL port (default 5432)
- Ensure user has CONNECT privilege

### Permission Errors
- Grant `pg_monitor` role (PostgreSQL 10+)
- For older versions, grant SELECT on specific system views

### High Query Execution Time
- Add indexes on frequently filtered columns
- Reduce collection frequency with `min_interval`
- Filter out unnecessary databases/tables

## References

- [PostgreSQL Documentation](https://www.postgresql.org/docs/)
- [PostgreSQL Statistics Collector](https://www.postgresql.org/docs/current/monitoring-stats.html)
- [SQL Exporter Configuration](https://github.com/burningalchemist/sql_exporter)

