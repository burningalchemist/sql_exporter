# Azure SQL Managed Instance Monitoring

This directory contains comprehensive monitoring configurations for Azure SQL Managed Instance, including collector configurations and Grafana dashboards.

## Overview

This example provides production-ready monitoring for Azure SQL Managed Instance with:
- **5 specialized collectors** for different performance aspects
- **Pre-built Grafana dashboard** with 6 panels
- **Best practices** for Azure SQL MI monitoring

## Files

### Collector Configurations

#### `mssql_mi_properties.collector.yml`
Instance properties and configuration:
- SQL Server version and edition
- Max server memory configuration
- Instance-level settings
- Collation and compatibility level

#### `mssql_mi_clerk.collector.yml`
Memory clerk statistics:
- Memory allocation by clerk type
- Buffer pool usage
- Plan cache sizes
- System memory pressure indicators

#### `mssql_mi_perf.collector.yml`
Performance counters:
- Batch requests/sec
- SQL compilations/sec
- Page life expectancy
- Buffer cache hit ratio
- Lock waits and deadlocks
- User connections

#### `mssql_mi_size.collector.yml`
Database and file sizing:
- Database sizes (data + log)
- File growth tracking
- Space utilization
- Autogrowth events

#### `mssql_mi_wait.collector.yml`
Wait statistics:
- Top wait types
- Wait time accumulation
- Signal wait time
- Resource wait time

### Dashboard

#### `grafana-dashboard/azure-sql-mi.json`
Complete Grafana dashboard with 6 panels:

1. **Overview Panel** (`overview.png`)
   - Instance health summary
   - Key performance metrics
   - Alert status

2. **CPU and Queuing** (`cpu-and-queuing.png`)
   - CPU utilization
   - Runnable task count
   - CPU queue length
   - Scheduler health

3. **Memory Panel** (`memory.png`)
   - Total server memory
   - Target server memory
   - Memory grants pending
   - Page life expectancy

4. **SQL Activity** (`sql-activity.png`)
   - Batch requests/sec
   - SQL compilations and recompilations
   - User connections
   - Active sessions

5. **Waits and Queues** (`waits-and-queues.png`)
   - Top wait types by time
   - Wait type trends
   - Resource contention indicators

6. **Log Activity** (`log-activity.png`)
   - Transaction log size
   - Log growth rate
   - Log flush wait time
   - Virtual log file count

## Quick Start

### 1. Create Database Secret

```bash
kubectl create secret generic azuresql-mi-credentials \
  --from-literal=dsn='username:password@myinstance.database.windows.net:1433/master?encrypt=true' \
  --namespace=monitoring
```

### 2. Deploy with Helm

```yaml
# values-azure-sql-mi.yaml
dynamicConfig:
  enabled: true
  type: sqlserver
  secretName: azuresql-mi-credentials
  secretKey: dsn
  useApplicationName: true
  template: |
    target:
      data_source_name: __TYPE__://__DSN__&app name=sql-exporter
      collectors:
        - mssql_mi_properties
        - mssql_mi_clerk
        - mssql_mi_perf
        - mssql_mi_size
        - mssql_mi_wait

# Include collector files via collectorFiles or inline in template
collectorFiles:
  mssql_mi_properties.collector.yml: |
    # Contents from file
  mssql_mi_clerk.collector.yml: |
    # Contents from file
  # ... other collectors

serviceMonitor:
  enabled: true
  interval: 30s
```

```bash
helm install sql-exporter ../../helm -f values-azure-sql-mi.yaml
```

### 3. Import Grafana Dashboard

1. Open Grafana UI
2. Navigate to **Dashboards** → **Import**
3. Upload `grafana-dashboard/azure-sql-mi.json`
4. Select your Prometheus datasource
5. Click **Import**

## Azure SQL MI Connection

### Connection String Format
```
sqlserver://username:password@instance-name.database.windows.net:1433?database=master&encrypt=true
```

### Required Parameters
- `encrypt=true` - **Required** for Azure SQL (TLS encryption)
- `database=master` - Connect to master for instance-level queries
- `app name=sql-exporter` - Identifier in `sys.dm_exec_sessions`

### Authentication Options

#### SQL Authentication (Recommended)
```
sqlserver://sqladmin:SecureP@ssw0rd@myinstance.database.windows.net:1433?database=master&encrypt=true
```

#### Azure AD Authentication
```
sqlserver://myuser@domain.com:password@myinstance.database.windows.net:1433?database=master&encrypt=true&fedauth=ActiveDirectoryPassword
```

## Permissions Setup

### Create Monitoring User

```sql
-- In master database
CREATE LOGIN sql_exporter WITH PASSWORD = 'SecurePassword123!';

-- Grant server-level permissions
GRANT VIEW SERVER STATE TO sql_exporter;
GRANT VIEW ANY DEFINITION TO sql_exporter;

-- For each monitored database
USE [YourDatabase];
CREATE USER sql_exporter FOR LOGIN sql_exporter;
GRANT VIEW DATABASE STATE TO sql_exporter;
```

### Verify Permissions

```sql
-- Check granted permissions
SELECT * FROM sys.server_permissions 
WHERE grantee_principal_id = USER_ID('sql_exporter');

-- Test DMV access
SELECT COUNT(*) FROM sys.dm_os_performance_counters;
SELECT COUNT(*) FROM sys.dm_os_wait_stats;
```

## Grafana Dashboard Configuration

### Add Dashboard Variables

The dashboard supports these variables:
- `$instance` - SQL MI instance name
- `$database` - Database filter
- `$interval` - Scrape interval

### Configure Alerts

Recommended alert thresholds:

1. **CPU > 80%** for 5 minutes
   ```promql
   avg(mssql_cpu_usage_percent{instance="$instance"}) > 80
   ```

2. **Page Life Expectancy < 300 seconds**
   ```promql
   mssql_page_life_expectancy{instance="$instance"} < 300
   ```

3. **Deadlocks detected**
   ```promql
   rate(mssql_deadlocks_total{instance="$instance"}[5m]) > 0
   ```

4. **Log size > 80% of limit**
   ```promql
   (mssql_log_size_mb / mssql_log_size_limit_mb) > 0.8
   ```

## Performance Tuning

### Collector Intervals

Adjust `min_interval` based on your needs:

```yaml
collectors:
  - collector_name: mssql_mi_properties
    min_interval: 300s  # Static data, check every 5 minutes
    
  - collector_name: mssql_mi_perf
    min_interval: 15s   # Performance counters, frequent updates
    
  - collector_name: mssql_mi_wait
    min_interval: 30s   # Wait stats, moderate frequency
    
  - collector_name: mssql_mi_size
    min_interval: 60s   # Database sizes, check every minute
```

### Query Optimization

All queries use DMVs which are already optimized. Key practices:
- Use `WITH (NOLOCK)` where appropriate
- Filter unnecessary databases: `WHERE database_id > 4`
- Limit result sets: `TOP 20` for wait stats

## Troubleshooting

### Connection Issues

**Error:** "Cannot open server 'xxx' requested by the login"
- Verify instance name is correct
- Check Azure SQL MI firewall rules
- Ensure VNet connectivity if using private endpoint

**Error:** "Login failed for user 'sql_exporter'"
- Verify user exists: `SELECT * FROM sys.sql_logins WHERE name = 'sql_exporter';`
- Check Azure AD authentication if using AAD
- Verify password hasn't expired

### Missing Metrics

**Some panels show "No data"**
1. Check collector is enabled in configuration
2. Verify permissions: `GRANT VIEW SERVER STATE`
3. Check Prometheus logs for scrape errors
4. Verify ServiceMonitor is discovered by Prometheus

**Dashboard variables not populating**
- Ensure `instance` label exists in metrics
- Check Prometheus datasource connection
- Verify metric names match dashboard queries

### High Resource Usage

**SQL MI shows high CPU from monitoring**
- Increase `min_interval` for collectors
- Reduce number of wait stats returned (TOP 10 instead of TOP 20)
- Disable less critical collectors during peak hours

## Azure SQL MI Specifics

### Differences from On-Premises SQL Server

- **No xp_cmdshell** - Some administrative queries unavailable
- **Limited sys tables** - Some system tables restricted
- **Automatic tuning** - Azure manages many settings automatically
- **Resource governance** - vCore and storage limits apply

### Azure-Specific Metrics

Consider adding:

```yaml
- metric_name: azure_sql_mi_storage_percent
  type: gauge
  help: 'Storage utilization percentage'
  query: |
    SELECT 
      (SUM(CAST(FILEPROPERTY(name, 'SpaceUsed') AS bigint)) * 8192.0 / 
       SUM(size) * 8192.0) * 100 as storage_percent
    FROM sys.database_files;
```

## Cost Optimization

- Set appropriate retention in Prometheus (e.g., 30 days)
- Use recording rules for frequently queried metrics
- Adjust scrape intervals based on SLA requirements
- Consider using Azure Monitor for native metrics alongside SQL Exporter

## References

- [Azure SQL MI Documentation](https://docs.microsoft.com/en-us/azure/azure-sql/managed-instance/)
- [SQL MI Monitoring Best Practices](https://docs.microsoft.com/en-us/azure/azure-sql/managed-instance/monitoring-with-dmvs)
- [DMV Reference](https://docs.microsoft.com/en-us/sql/relational-databases/system-dynamic-management-views/)
- [Grafana Provisioning](https://grafana.com/docs/grafana/latest/administration/provisioning/)

