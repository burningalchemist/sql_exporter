# Command-line parameters

| Property                           | Default            | Description                                                                      |
| ---------------------------------- | ------------------ | -------------------------------------------------------------------------------- |
| **--web.listen-address**           | `:9399`            | Address to listen on for web interface and telemetry                             |
| **--web.metrics-path**             | `/metrics`         | Path under which to expose metrics                                               |
| **--web.enable-reload**            | `false`            | Enable reload collector data handler                                             |
| **--web.config.file**              |                    | TLS/BasicAuth configuration file path                                            |
| **--config.file**                  | `sql_exporter.yml` | SQL Exporter configuration file path (overridden by `SQLEXPORTER_CONFIG` if set) |
| **--config.check**                 | `false`            | Check the configuration file and exit                                            |
| **--config.enable-ping**           | `true`             | Enable ping for targets                                                          |
| **--config.ignore-missing-values** | `false`            | Ignore results with missing values for the requested columns                     |
| **--config.data-source-name**      |                    | Override the target DSN from the configuration file (single target mode only)    |
| **--config.target-label**          | `target`           | Name of the label that holds the target name                                     |
| **--log.format**                   | `logfmt`           | Set log output format (`logfmt \| json`)                                         |
| **--log.level**                    | `info`             | Set log level (`debug\|info\|warn\|error`)                                       |
| **--log.file**                     |                    | Log file to write to, leave empty to write to `stderr`                           |
| **--version**                      | `false`            | Print version information and exit                                               |
