# Scraping Special Backends

## PgBouncer

Given that PgBouncer is a connection pooler, it doesn't support all the commands that a regular SQL database does, so
we need to make some adjustments to the configuration:

- add `enable_ping: false` to the target or job configuration as PgBouncer doesn't support the ping command;
- add `no_prepared_statement: true` to the metric or query configuration as PgBouncer doesn't support the extended query
  protocol;

For libpq (postgres) driver we only need to set `no_prepared_statement: true` parameter. For pgx driver, we also need
to add `default_query_exec_mode=simple_protocol` parameter to the DSN (for v5).

Below is an example of a metric configuration for PgBouncer:

```yaml
metrics:
  - metric_name: max_connections
    no_prepared_statement: true
    type: gauge
    help: 'Maximum number of allowed server connections for the database.'
    values: [max_connections]
    key_labels:
      - name
      - database
      - force_user
      - pool_mode
      - disabled
      - paused
      - current_connections
      - reserve_pool
      - min_pool_size
      - pool_size
      - port
    query: |
      SHOW DATABASES;
```

## ProxySQL, Clickhouse

Same goes for ProxySQL and Clickhouse, where we need to add `no_prepared_statement: true` to the metric or query
configuration, as these databases do not support prepared statements.

## Snowflake

In case, you connect to a data warehouse (e.g. Snowflake) you don't want to keep online all the time (due to the extra
cost), you might want to disable `ping` by setting `enable_ping: false`.
