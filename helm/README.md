# sql-exporter

![Version: 0.2.1](https://img.shields.io/badge/Version-0.2.1-informational?style=flat-square) ![Type: application](https://img.shields.io/badge/Type-application-informational?style=flat-square) ![AppVersion: 0.13.0](https://img.shields.io/badge/AppVersion-0.13.0-informational?style=flat-square)

Database agnostic SQL exporter for Prometheus

## Source Code

* <https://github.com/burningalchemist/sql_exporter>

## Maintainers

| Name | Email | Url |
| ---- | ------ | --- |
| Nikolai Rodionov | <allanger@zohomail.com> | <https://badhouseplants.net> |

## Installing the Chart

To install the chart with the release name `sql-exporter`:

```console
helm repo add sql_exporter https://burningalchemist.github.io/sql_exporter/
helm install sql_exporter/sql-exporter
```

## Chart Values

### General parameters

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| nameOverride | string | `""` | Provide a name in place of `sql-exporter` |
| fullnameOverride | string | `""` | String to fully override "sql-exporter.fullname" |
| image.repository | string | `"burningalchemist/sql_exporter"` | Image repository |
| image.pullPolicy | string | `"IfNotPresent"` | Image pull policy |
| image.tag | string | `appVersion` value from `Chart.yaml` | Image tag |
| imagePullSecrets | list | `[]` | Secrets with credentials to pull images from a private registry |
| service.type | string | `"ClusterIP"` | Service type |
| service.port | int | `80` | Service port |
| service.labels | object | `{}` | Service labels |
| service.annotations | object | `{}` | Service annotations |
| resources | object | `{}` | Resource limits and requests for the application controller pods |
| podLabels | object | `{}` | Pod labels |
| podAnnotations | object | `{}` | Pod annotations |
| podSecurityContext | object | `{}` | Pod security context |

### Prometheus ServiceMonitor

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| serviceMonitor.enabled | bool | `true` | Enable ServiceMonitor |
| serviceMonitor.interval | string | `"15s"` | ServiceMonitor interval |
| serviceMonitor.path | string | `"/metrics"` | ServiceMonitor path |
| serviceMonitor.scrapeTimeout | string | `"60s"` | ServiceMonitor scrape timeout |

### Configuration

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| config.global.scrape_timeout | string | `"10s"` | Scrape timeout |
| config.global.scrape_timeout_offset | string | `"500ms"` | Scrape timeout offset. Must be strictly positive. |
| config.global.min_interval | string | `"0s"` | Minimum interval between collector runs. |
| config.global.max_connections | int | `3` | Number of open connections. |
| config.global.max_idle_connections | int | `3` | Number of idle connections. |
| target | object | `nil` | Check documentation. Mutually exclusive with `jobs`  |
| jobs   | list | `nil` | Check documentation. Mutually exclusive with `target` |
| collector_files | list | `[]` | Check documentation |

To configure `target`, `jobs`, `collector_files` please refer to the [documentation](https://github.com/burningalchemist/sql_exporter/blob/master/documentation/sql_exporter.yml) in the source repository. These values are not set by default.

It's also possible to define collectors (i.e. metrics and queries) in separate files, and specify the filenames in the `collector_files` list. For that we can use `CollectorFiles` field (check `values.yaml` for the available example).
