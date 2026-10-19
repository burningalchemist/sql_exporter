# Environment Variables

Here is a list of available environment variables that can be used to configure SQL Exporter:

| Environment Variable          | Default            | Description                                                         |
| :---------------------------- | ------------------ | :------------------------------------------------------------------ |
| `SQLEXPORTER_CONFIG`          | `sql_exporter.yml` | file path to the configuration file                                 |
| `SQLEXPORTER_COLLECTOR_FILES` |                    | glob pattern(s) for collector definition files, semicolon-separated |

| Environment Variable                            | Default | Description                                                           |
| :---------------------------------------------- | ------- | :-------------------------------------------------------------------- |
| `SQLEXPORTER_GLOBAL_MIN_INTERVAL`               | `0s`    | minimum interval between query executions                             |
| `SQLEXPORTER_GLOBAL_SCRAPE_TIMEOUT`             | `10s`   | per-scrape timeout, global                                            |
| `SQLEXPORTER_GLOBAL_SCRAPE_TIMEOUT_OFFSET`      | `500ms` | offset to subtract from timeout in seconds                            |
| `SQLEXPORTER_GLOBAL_SCRAPE_ERROR_DROP_INTERVAL` | `0s`    | interval to drop scrape errors from the error counter                 |
| `SQLEXPORTER_GLOBAL_PING_INTERVAL`              | `0s`    | interval between database pings                                       |
| `SQLEXPORTER_GLOBAL_MAX_CONNECTIONS`            | `3`     | maximum number of open connections to any one target                  |
| `SQLEXPORTER_GLOBAL_MAX_IDLE_CONNECTIONS`       | `3`     | maximum number of idle connections to any one target                  |
| `SQLEXPORTER_GLOBAL_MAX_CONNECTION_LIFETIME`    | `0`     | maximum amount of time a connection may be reused to any one target   |
| `SQLEXPORTER_GLOBAL_WARMUP_DELAY`               | `0s`    | delay between executing collectors during cache population at startup |
| `SQLEXPORTER_GLOBAL_ENABLE_QUERY_METRICS`       | `false` | expose per-query duration and row count metrics                       |

| Environment Variable             | Default | Description                                                                              |
| :------------------------------- | ------- | :--------------------------------------------------------------------------------------- |
| `SQLEXPORTER_TARGET_NAME`        |         | (Optional) Target name, generates `up` and `scrape_duration_seconds` additional metrics) |
| `SQLEXPORTER_TARGET_DSN`         |         | URL-format connection string for the target database                                     |
| `SQLEXPORTER_TARGET_COLLECTORS`  |         | Collector names to execute on the target, should be semicolon-separated                  |
| `SQLEXPORTER_TARGET_ENABLE_PING` | `true`  | ping the database before executing collectors                                            |
