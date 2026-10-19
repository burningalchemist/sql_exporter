# TLS and Basic Auth

SQL Exporter supports TLS and Basic Authentication. This enables better control of the various HTTP endpoints.

To use TLS and/or Basic Authentication, you need to pass a configuration file using the `--web.config.file` parameter.

The format of the file is described in the
[exporter-toolkit](https://github.com/prometheus/exporter-toolkit/blob/master/docs/web-configuration.md) repository.
Also please check the available
[example](https://github.com/prometheus/exporter-toolkit/blob/master/docs/web-config.yml).
