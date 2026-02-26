# Microsoft SQL Server Collector Examples

This directory contains example collector configurations for Microsoft SQL Server and Azure SQL Database.

## Overview

These collectors demonstrate common SQL Server monitoring patterns using `sql_exporter`. They cover essential performance metrics, resource utilization, and health indicators.

## Files

### `mssql_standard.collector.yml`
Standard SQL Server metrics including:
- Database sizes and file growth
- Buffer cache hit ratios
- Page life expectancy
- Batch requests and compilations per second
- User connections
- Lock waits and deadlocks
- Transaction log usage
- Backup status and age

### `sql_exporter.yml`
Complete configuration example with connection settings.

## Usage with Helm Chart

### Method 1: Using `collectorFiles`

```yaml
collectorFiles:
  mssql_standard.collector.yml: |
    # Contents of mssql_standard.collector.yml

config:
  target:
    data_source_name: "sqlserver://username:password@hostname:1433?database=master"
  collector_files:
    - "*.collector.yml"
```

### Method 2: Using Dynamic Config with Secret

```yaml
dynamicConfig:
  enabled: true
  type: sqlserver
  secretName: mssql-credentials
  secretKey: dsn
  useApplicationName: true
  template: |
    target:
      data_source_name: __TYPE__://__DSN__?database=master&app name=sql-exporter
      collectors:
        - mssql_standard
    # Include collector definitions
```

## Connection String Formats

### SQL Server Authentication
```
sqlserver://username:password@hostname:1433?database=master
```

### Windows Authentication
```
sqlserver://hostname:1433?database=master&integrated security=SSPI
```

### Azure SQL Database
```
sqlserver://username:password@server.database.windows.net:1433?database=dbname&encrypt=true
```

### Connection Parameters
- `database` - Initial database to connect to
- `app name` - Application name visible in `sys.dm_exec_sessions`
- `encrypt` - Enable encryption (required for Azure SQL)
- `trust server certificate` - Skip certificate validation (for self-signed certs)
- `connection timeout` - Connection timeout in seconds

## Common Customizations

### Filter Specific Databases
Exclude system databases from size calculations:

```sql
WHERE database_id > 4  -- Skip master, tempdb, model, msdb
  AND state_desc = 'ONLINE'
```

### Add Wait Statistics
Monitor specific wait types:

```yaml
collectors:
  - collector_name: mssql_waits
    metrics:
      - metric_name: mssql_wait_time_ms
        type: counter
        help: 'Wait time by wait type'
        key_labels:
          - wait_type
        values:
          - wait_time_ms
        query: |
          SELECT 
            wait_type,
            wait_time_ms
          FROM sys.dm_os_wait_stats
          WHERE wait_type NOT LIKE 'SLEEP%'
            AND wait_type NOT LIKE 'BROKER%'
          ORDER BY wait_time_ms DESC;
```

### Monitor Always On Availability Groups
```yaml
collectors:
  - collector_name: mssql_availability_groups
    metrics:
      - metric_name: mssql_ag_replica_health
        type: gauge
        help: 'Always On AG replica health'
        key_labels:
          - ag_name
          - replica_server_name
          - synchronization_state
        values:
          - is_primary_replica
        query: |
          SELECT 
            ag.name as ag_name,
            ar.replica_server_name,
            rs.synchronization_state_desc as synchronization_state,
            CASE WHEN ar.replica_server_name = @@SERVERNAME 
              AND rs.role_desc = 'PRIMARY' THEN 1 ELSE 0 END as is_primary_replica
          FROM sys.dm_hadr_availability_replica_states rs
          JOIN sys.availability_replicas ar ON rs.replica_id = ar.replica_id
          JOIN sys.availability_groups ag ON ag.group_id = ar.group_id;
```

## Permissions Required

### Minimum Permissions
```sql
-- Create login and user
CREATE LOGIN sql_exporter WITH PASSWORD = 'SecurePassword123!';
CREATE USER sql_exporter FOR LOGIN sql_exporter;

-- Grant VIEW SERVER STATE permission
GRANT VIEW SERVER STATE TO sql_exporter;
GRANT VIEW ANY DEFINITION TO sql_exporter;

-- For database-specific metrics
USE [YourDatabase];
CREATE USER sql_exporter FOR LOGIN sql_exporter;
GRANT VIEW DATABASE STATE TO sql_exporter;
```

### Azure SQL Database
```sql
-- In master database
CREATE LOGIN sql_exporter WITH PASSWORD = 'SecurePassword123!';

-- In each monitored database
CREATE USER sql_exporter FOR LOGIN sql_exporter;
GRANT VIEW DATABASE STATE TO sql_exporter;
```

## Grafana Dashboards

Compatible with:
- [SQL Server Dashboard](https://grafana.com/grafana/dashboards/409)
- [SQL Server Performance](https://grafana.com/grafana/dashboards/11176)

## Testing

Test the connection and queries:

```bash
# Using sqlcmd (if available in pod)
sqlcmd -S hostname -U sql_exporter -P 'password' -d master -Q "SELECT @@VERSION;"

# Test from outside
sqlcmd -S hostname,1433 -U sql_exporter -P 'password' -Q "SELECT * FROM sys.dm_os_performance_counters WHERE counter_name = 'Page life expectancy';"
```

## Performance Considerations

### Minimize Impact
- Use `NOLOCK` hint for read-only queries (where appropriate)
- Set appropriate `min_interval` (e.g., 30s-60s)
- Avoid queries on large tables without proper indexes
- Use DMVs (Dynamic Management Views) instead of querying user tables

### Example with NOLOCK
```sql
SELECT database_id, name, state_desc
FROM sys.databases WITH (NOLOCK)
WHERE state_desc = 'ONLINE';
```

## Troubleshooting

### Connection Issues
**Error:** "Login failed for user"
- Verify SQL authentication is enabled: `sp_configure 'remote admin connections', 1;`
- Check firewall rules allow port 1433
- For Azure SQL: Add client IP to firewall rules

**Error:** "SSL Security error"
- For self-signed certificates: Add `trust server certificate=true`
- For Azure SQL: Ensure `encrypt=true` is in connection string

### Permission Errors
**Error:** "The SELECT permission was denied on the object 'sys.dm_os_performance_counters'"
- Grant VIEW SERVER STATE permission
- Verify user exists in master database

### High CPU Usage
- Increase `min_interval` to reduce query frequency
- Simplify complex queries
- Add indexes on filtered columns in DMVs (usually not needed)

## SQL Server Versions Supported

- SQL Server 2012 and later
- Azure SQL Database
- Azure SQL Managed Instance
- SQL Server on Linux

## References

- [SQL Server DMVs Documentation](https://docs.microsoft.com/en-us/sql/relational-databases/system-dynamic-management-views/)
- [Azure SQL Monitoring](https://docs.microsoft.com/en-us/azure/azure-sql/database/monitoring-with-dmvs)
- [SQL Exporter for SQL Server](https://github.com/burningalchemist/sql_exporter)

