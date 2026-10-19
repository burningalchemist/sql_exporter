# Command-line parameters

| Property                           | Default            | Description                                                  |
| ---------------------------------- | ------------------ | ------------------------------------------------------------ |
| **--web.listen-address**           | `:9399`            | Address to listen on for web interface and telemetry         |
| **--web.metrics-path**             | `/metrics`         | Path under which to expose metrics                           |
| **--web.enable-reload**            | `false`            | Enable reload collector data handler                         |
| **--web.config.file**              |                    | TLS/BasicAuth configuration file path                        |
| **--config.file**                  | `sql_exporter.yml` | SQL Exporter configuration file path                         |
| **--config.enable-ping**           | `true`             | Enable ping for targets                                      |
| **--config.ignore-missing-values** | `false`            | Ignore results with missing values for the requested columns |
| **--log.format**                   | `logfmt`           | Set log output format (`logfmt \| json`)                     |
| **--log.level**                    | `info`             | Set log level (`debug\|info\|warn\|error`)                   |
| **--log.file**                     |                    | Log file to write to, leave empty to write to `stderr`       |
