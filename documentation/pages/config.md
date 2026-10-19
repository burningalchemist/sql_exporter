# Reference

## Global

| Property                       | Default | Description                                                                                                                                                                                                                                                                                                                                                                            |
| ------------------------------ | ------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **scrape_timeout**             | `10s`   | Scrape timeouts ensure that scraping completes in reasonable time, and slow queries are canceled early when the database is already under heavy load                                                                                                                                                                                                                                   |
| **scrape_timeout_offset**      | `500ms` | Offset subtracted from Prometheus `scrape_timeout` to give us some headroom and prevent Prometheus from timing out first. Must be strictly positive.                                                                                                                                                                                                                                   |
| **scrape_error_drop_interval** | `0s`    | Interval between dropping `scrape_errors_total` metric. By default metrics are persistent.                                                                                                                                                                                                                                                                                             |
| **min_interval**               | `0s`    | Minimum interval between collector runs. By default collectors are executed on every scrape. Essentially this is a cache TTL for collector results, but it also applies to the first scrape after startup, so it can be used to stagger expensive collector runs and avoid overloading the database.                                                                                   |
| **warmup_delay**               | `0s`    | Delay between collector scrapes during the startup cache warmup. This can be used to avoid overloading the database with expensive queries (thundering herd) when the cache is empty and all collectors are executed at once. It's intended to be used together with 'min_interval' to stagger collector runs. By default all collectors are executed immediately on the first scrape. |
| **max_connections**            | `3`     | Maximum number of open connections to any one target. Metric queries will run concurrently on multiple connections, as will concurrent scrapes.                                                                                                                                                                                                                                        |
| **max_idle_connections**       | `3`     | Maximum number of idle connections to any one target. Unless you use very long collection intervals, this should always be the same as `max_connections`.                                                                                                                                                                                                                              |
| **max_connection_lifetime**    | `0s`    | Maximum amount of time a connection may be reused to any one target                                                                                                                                                                                                                                                                                                                    |

### Timeouts

Prometheus informs targets of its own scrape timeout (via the `X-Prometheus-Scrape-Timeout-Seconds` request header)
so the actual timeout is computed as:

```text
min(scrape_timeout, X-Prometheus-Scrape-Timeout-Seconds - scrape_timeout_offset)
```

If `scrape_timeout` ≤ 0, no timeout is set unless Prometheus provides one.

### Connections

TBD

## Target (Single target mode)

The target to monitor and the collectors to execute on it.

| Property             | Type           | Description                                                                                                                                                                                                                        |
| -------------------- | -------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **name**             | `String?`      | (Optional) Target name. Setting this field enables extra metrics e.g. `up` and `scrape_duration` with the `target` label that are always returned on a scrape.                                                                     |
| **data_source_name** | `String`       | Data source name always has a URI schema that matches the driver name. In some cases (e.g. MySQL) the schema gets dropped or replaced to match the driver expected DSN format.                                                     |
| **collectors**       | `List(String)` | Collectors (referenced by name) to execute on the target.                                                                                                                                                                          |
| **enable_ping**      | `Boolean`      | In case you need to connect to a backend that only responds to a limited set of commands (e.g. pgbouncer) or a data warehouse you don't want to keep online all the time (due to the extra cost), you might want to disable `ping` |

::: details Click to expand - Single target mode example

```yaml
target:
  name: "prices_db"
  data_source_name: "sqlserver://prom_user:prom_password@dbserver1.example.com:1433"
  collectors: [pricing_data_freshness, pricing_*]
  enable_ping: true
```

:::

## Jobs (Multiple target mode)

| Property           | Type                 | Description                                                                                                                                                                                                                        |
| ------------------ | -------------------- | ---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **job_name**       | `String`             | Job name. Setting this field enables extra metrics e.g. `up` and `scrape_duration` with the `target` label that are always returned on a scrape.                                                                                   |
| **static_configs** | `List(StaticConfig)` | Collections of statically defined targets                                                                                                                                                                                          |
| **collectors**     | `List(String)`       | Collectors (referenced by name) to execute on the targets                                                                                                                                                                          |
| **enable_ping**    | `Boolean`            | In case you need to connect to a backend that only responds to a limited set of commands (e.g. pgbouncer) or a data warehouse you don't want to keep online all the time (due to the extra cost), you might want to disable `ping` |

### Static Configs

| Property    | Type          | Description                                               |
| ----------- | ------------- | --------------------------------------------------------- |
| **targets** | `Map(String)` | Map of unique target names and their connection strings   |
| **labels**  | `Map(String)` | Labels to apply to all metrics collected from the targets |

::: details Click to expand - Multiple target mode example

```yaml
jobs:
  - job_name: db_targets
    collectors: [pricing_data_freshness, pricing_*]
    enable_ping: true
    static_configs:
      - targets:
          pg1: "pg://db1@127.0.0.1:25432/postgres?sslmode=disable"
          pg2: "postgresql://username:password@pg-host.example.com:5432/dbname?sslmode=disable"
        labels: # Optional, arbitrary key/value pair for all targets
          cluster: cluster1
```

:::

## Collectors

A collector is a named set of related metrics that are collected together. It can be referenced by name, possibly along
with other collectors.

Collectors may be defined in two inclusive ways:

1. inline in the exporter configuration file, under the `collectors` field;
2. as separate files (one collector per file), and referenced in the `collector_files` of the exporter configuration
   file, making them easy to share and reuse.

| Property           | Type               | Description                                                                                                                                                    |
| ------------------ | ------------------ | -------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| **collector_name** | `String?`          | (Optional) Target name. Setting this field enables extra metrics e.g. `up` and `scrape_duration` with the `target` label that are always returned on a scrape. |
| **min_interval**   | `String`           | Similar to `global.min_interval`, but applies to this collector only.                                                                                          |
| **metrics**        | `List(MetricSpec)` | List of Metric definitions                                                                                                                                     |
| **queries**        | `List(QuerySpec)?` | (Optional) List of Query definitions                                                                                                                           |

### Metric

A metric is a Prometheus metric with name, type, help text and (optional) additional labels, paired with exactly
one query to populate the metric labels and values from.

| Property                  | Type               | Description                                                                                                                        |
| ------------------------- | ------------------ | ---------------------------------------------------------------------------------------------------------------------------------- |
| **metric_name**           | `String`           | The metric name, type and help text, as exported to /metrics                                                                       |
| **type**                  | `(counter\|gauge)` | Underlying metric type so Prometheus and monitoring tools know how to interpret and query the data                                 |
| **help**                  | `String`           | Human-readable description or documentation string for a metric                                                                    |
| **query**                 | `String`           | SQL query to execute.                                                                                                              |
| **key_labels**            | `Set(String)?`     | Set of labels derived from the existing result set columns                                                                         |
| **values**                | `List(String)`     | Column name(s) to return as a metric value. Can be multiple but requires `value_label`                                             |
| **value_label**           | `Map(String)?`     | Label populated with the value column name, configured via `values`. Required when multiple value columns are configured           |
| **static_labels**         | `Map(String)?`     | Arbitrary key/value pairs                                                                                                          |
| **static_value**          | `Integer`          | Arbitraty static value. Useful, when not the value matters, but the string data (populated as labels)                              |
| **timestamp_value**       | `String?`          | Timestamp column name to return a metric with an explicit timestamp                                                                |
| **no_prepared_statement** | `Boolean`          | Do not use a prepared statement if `true`. Might be required to execute queries against some backends (e.g. Clickhouse, PGBouncer) |
| **query_ref**             | `String?`          | Query name (referenced by name)                                                                                                    |

The result columns conceptually fall into two categories:

- zero or more `key_labels` columns: their values will be directly mapped to labels of the same name;
- one or more `values` columns:
  - if exactly one value column, the column name is ignored and its value becomes the metric value;
  - with multiple value columns, a `value_label` **must be** defined; the column name will populate this label and
    the column value will populate the metric value.

### Queries

Named queries, referenced by one or more metrics, through `query_ref`.

| Property                  | Type      | Description                                                                                                                        |
| ------------------------- | --------- | ---------------------------------------------------------------------------------------------------------------------------------- |
| **query_name**            | `String`  | The query name to be used as a reference                                                                                           |
| **query**                 | `String`  | SQL query to execute.                                                                                                              |
| **no_prepared_statement** | `Boolean` | Do not use a prepared statement if `true`. Might be required to execute queries against some backends (e.g. Clickhouse, PGBouncer) |

::: details Click to expand - Collector example

```yaml
collector_name: pricing_data_freshness

metrics:
  - metric_name: pricing_update_time
    type: gauge
    help: "Time when prices for a market were last updated."
    key_labels:
      - Market
    static_labels:
      portfolio: income
    values: [LastUpdateTime]
    query: |
      SELECT Market, max(UpdateTime) AS LastUpdateTime
      FROM MarketPrices
      GROUP BY Market
```

:::

## Collector Files

> [!NOTE]
> The `collectors` and `collector_files` configurations also support [Glob pattern matching](https://pkg.go.dev/path/filepath#Match).
> To match names with literal pattern terms in them, e.g. `collector_*1*`, these must be escaped: `collector_\*1\*`.

| Property            | Type           | Description                                                                   |
| ------------------- | -------------- | ----------------------------------------------------------------------------- |
| **collector_files** | `List(String)` | Collector files specifies a list of globs. One collector definition per file. |



## Complete Example

::: details Click to expand **`./sql_exporter.yml`**:

```yaml
global:
  scrape_timeout: 10s
  scrape_timeout_offset: 500ms
  scrape_error_drop_interval: 0s
  min_interval: 0s
  warmup_delay: 0s
  max_connections: 3
  max_idle_connections: 3

target:
  name: mssql_database
  data_source_name: "sqlserver://prom_user:prom_password@dbserver1.example.com:1433/dbname"
  collectors: [mssql_standard]
  enable_ping: true

collectors:
  - collector_name: mssql_standard
    metrics:
      # The metric name, type and help text, as exported to /metrics.
      - metric_name: mssql_log_growths
        type: counter
        help: "Total number of times the transaction log has been expanded since last restart, per database."
        key_labels:
          - db
        static_labels:
          env: dev
          region: europe
        values: [counter]
        query: |
          SELECT rtrim(instance_name) AS db, cntr_value AS counter
          FROM sys.dm_os_performance_counters
          WHERE counter_name = 'Log Growths' AND instance_name <> '_Total'

      # A different metric, with multiple values produced from each result row.
      - metric_name: mssql_io_stall_seconds
        type: counter
        help: "Stall time in seconds per database and I/O operation."
        key_labels:
          - db
        value_label: operation
        values:
          - io_stall_read
          - io_stall_write
        query_ref: io_stall

      # Another metric, uses same named query (referenced through query_ref) as mssql_io_stall_seconds.
      - metric_name: mssql_io_stall_total_seconds
        type: counter
        help: "Total stall time in seconds per database."
        key_labels:
          - db
        values:
          - io_stall
        query_ref: io_stall

      # Metric with a static value to retrieve string data.
      - metric_name: mssql_hostname
        type: gauge
        help: "Database server hostname"
        key_labels:
          - hostname
        static_value: 1
        query: |
          SELECT @@SERVERNAME AS hostname

    # Named queries, referenced by one or more metrics, through query_ref.
    queries:
      - query_name: io_stall
        query: |
          SELECT
            cast(DB_Name(a.database_id) as varchar) AS db,
            sum(io_stall_read_ms) / 1000.0 AS io_stall_read,
            sum(io_stall_write_ms) / 1000.0 AS io_stall_write,
            sum(io_stall) / 1000.0 AS io_stall
          FROM
            sys.dm_io_virtual_file_stats(null, null) a
          INNER JOIN sys.master_files b ON a.database_id = b.database_id AND a.file_id = b.file_id
          GROUP BY a.database_id

collector_files:
  - "*.collector.yml"
```

:::
