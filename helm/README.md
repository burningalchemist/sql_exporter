# sql-exporter

![Version: 0.16.2](https://img.shields.io/badge/Version-0.16.2-informational?style=flat-square) ![Type: application](https://img.shields.io/badge/Type-application-informational?style=flat-square) ![AppVersion: 0.20.0](https://img.shields.io/badge/AppVersion-0.20.0-informational?style=flat-square)

Database-agnostic SQL exporter for Prometheus

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

### Ingress support

It's possible to enable the ingress creation by setting

```yaml
#Values
ingress:
  enabled: true
```

But as the sql_operator has a direct connection to databases,
it might expose the database servers to possible DDoS attacks.
It's not recommended by maintainers to use ingress for accessing the exporter,
but if there are no other options,
security measures should be taken.

For example, a user might enable the basic auth on the ingress level.
Take a look on how it's done at the
[nginx ingress controller](https://kubernetes.github.io/ingress-nginx/examples/auth/basic/)
as an example.

## Security Features

This chart supports TLS/HTTPS encryption and basic authentication for the metrics endpoint:

- **TLS Encryption**: Configure `webConfig.tls.secretName` to enable HTTPS with TLS 1.3 support and configurable cipher suites
- **Basic Authentication**: Set `webConfig.basicAuth.enabled` to protect metrics with bcrypt-hashed passwords (automatically hashed from plaintext secrets via init container)
- **Separate Secrets**: TLS certificates and authentication passwords can use the same secret or separate secrets for flexibility
- **ServiceMonitor Integration**: Prometheus ServiceMonitor automatically configures HTTPS and authentication when enabled

See the [examples directory](../examples/) for complete configuration examples: `tls-only`, `auth-only`, and `tls-auth`.

## Chart Values

### General parameters

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| commonAnnotations | object | `{}` | Common annotations to add to all the deployed resources |
| commonLabels | object | `{}` | Common labels to add to all deployed resources |
| createConfig | bool | `true` | Set to true to create a config as a part of the helm chart |
| extraContainers | object | `{}` | Arbitrary sidecar containers list |
| extraManifests | list | `[]` | Arbitrary manifests list |
| fullnameOverride | string | `""` | String to fully override "sql-exporter.fullname" |
| image.pullPolicy | string | `"IfNotPresent"` | Image pull policy |
| image.repository | string | `"burningalchemist/sql_exporter"` | Image repository |
| image.tag | string | `appVersion` value from `Chart.yaml` | Image tag |
| imagePullSecrets | list | `[]` | Secrets with credentials to pull images from a private registry |
| ingress.annotations | object | `{}` | Ingress annotations |
| ingress.enabled | bool | `false` |  |
| ingress.host | string | `""` | Ingress host |
| ingress.ingressClassName | string | `""` | Ingress class name |
| ingress.labels | object | `{}` | Ingress labels |
| ingress.path | string | `"/"` | Ingress path |
| ingress.tls | object | `{"crt":"","enabled":false,"key":"","secretName":""}` | Ingress TLS, can be defined by cert secret, or by key and cert. |
| ingress.tls.crt | string | `""` | Ingress tls.crt, required if you don't have secret name. |
| ingress.tls.key | string | `""` | Ingress tls.key, required if you don't have secret name. |
| ingress.tls.secretName | string | `""` | Ingress tls secret if already exists. |
| initContainers | object | `{}` | Arbitrary sidecar containers list for 1.29+ kubernetes |
| logFormat | string | `"logfmt"` | Set log format (logfmt if unset) |
| logLevel | string | `"info"` | Set log level (info if unset) |
| nameOverride | string | `""` | Provide a name in place of `sql-exporter` |
| podAnnotations | object | `{}` | Pod annotations |
| podLabels | object | `{}` | Pod labels |
| podSecurityContext | object | `{}` | Pod security context |
| reloadEnabled | bool | `false` | Enable reload collector data handler (endpoint /reload) |
| resources | object | `{}` | Resource limits and requests for the application controller pods |
| service.annotations | object | `{}` | Service annotations |
| service.labels | object | `{}` | Service labels |
| service.type | string | `"ClusterIP"` | Service type |
| serviceAccount.annotations | object | `{}` | Annotations to add to the Service Account |
| serviceAccount.create | bool | `true` | Specifies whether a Service Account should be created, creates "sql-exporter" service account if true, unless overriden. Otherwise, set to `default` if false, and custom service account name is not provided. Check all the available parameters. |
| webConfig | object | `{"basicAuth":{"bcryptCost":12,"enabled":false,"initFromSecret":{"enabled":false,"image":"httpd:alpine","imagePullPolicy":"IfNotPresent","secretKey":"password","secretName":""},"username":"prometheus","users":{}},"enabled":false,"template":"","tls":{"certFile":"tls.crt","certKey":"tls.crt","keyFile":"tls.key","keyKey":"tls.key","secretName":""}}` | Enable and configure Prometheus web config file support web-config.yml is automatically placed at /etc/sql_exporter/web-config.yml |
| webConfig.basicAuth | object | `{"bcryptCost":12,"enabled":false,"initFromSecret":{"enabled":false,"image":"httpd:alpine","imagePullPolicy":"IfNotPresent","secretKey":"password","secretName":""},"username":"prometheus","users":{}}` | Basic authentication configuration for web-config |
| webConfig.basicAuth.bcryptCost | int | `12` | Bcrypt cost used when hashing via initFromSecret |
| webConfig.basicAuth.enabled | bool | `false` | Enable basic auth in web-config; passwords must be bcrypt hashes |
| webConfig.basicAuth.initFromSecret | object | `{"enabled":false,"image":"httpd:alpine","imagePullPolicy":"IfNotPresent","secretKey":"password","secretName":""}` | Initialize basic auth from plaintext secret using bcrypt |
| webConfig.basicAuth.initFromSecret.enabled | bool | `false` | Use an initContainer to read plaintext from a secret and bcrypt it into web-config |
| webConfig.basicAuth.initFromSecret.image | string | `"httpd:alpine"` | Image used for bcrypt hashing (httpd:alpine has htpasswd at /usr/local/apache2/bin/htpasswd) |
| webConfig.basicAuth.initFromSecret.imagePullPolicy | string | `"IfNotPresent"` | Image pull policy for bcrypt hashing image |
| webConfig.basicAuth.initFromSecret.secretKey | string | `"password"` | Key in the secret that contains plaintext password |
| webConfig.basicAuth.initFromSecret.secretName | string | `""` | Secret name containing plaintext password |
| webConfig.basicAuth.username | string | `"prometheus"` | Username to protect /metrics |
| webConfig.basicAuth.users | object | `{}` | Map of username: bcryptHash (when not using initFromSecret) |
| webConfig.template | string | `""` | Template for web-config content (Exporter Toolkit format). Set to empty string to use default template (defined in _helpers.tpl) Default: TLS 1.3 with AES-GCM cipher suites, uses cert from webConfig.tls.secretName You can override with your own YAML string here if needed |
| webConfig.tls | object | `{"certFile":"tls.crt","certKey":"tls.crt","keyFile":"tls.key","keyKey":"tls.key","secretName":""}` | TLS configuration for web-config |
| webConfig.tls.certFile | string | `"tls.crt"` | Filename to project the certificate into the container |
| webConfig.tls.certKey | string | `"tls.crt"` | Key name within the secret for certificate |
| webConfig.tls.keyFile | string | `"tls.key"` | Filename to project the key into the container |
| webConfig.tls.keyKey | string | `"tls.key"` | Key name within the secret for key |
| webConfig.tls.secretName | string | `""` | Optional secret that holds tls.crt/tls.key. When set, it is mounted and used by web-config. |

### Prometheus ServiceMonitor

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| serviceMonitor.enabled | bool | `true` | Enable ServiceMonitor |
| serviceMonitor.interval | string | `"15s"` | ServiceMonitor interval |
| serviceMonitor.metricRelabelings | object | `{}` | ServiceMonitor metric relabelings |
| serviceMonitor.namespace | string | `nil` | ServiceMonitor namespace override (default is .Release.Namespace) |
| serviceMonitor.path | string | `"/metrics"` | ServiceMonitor path |
| serviceMonitor.relabelings | object | `{}` | ServiceMonitor relabelings |
| serviceMonitor.selector | object | `{}` | Additional labels for ServiceMonitor (for Prometheus serviceMonitorSelector matching) Example: selector: { monitored: dox-prometheus } |
| serviceMonitor.scrapeTimeout | string | `nil` | ServiceMonitor scrape timeout |

### Configuration

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| config | object | `{"global":{"max_connections":3,"max_idle_connections":3,"min_interval":"0s","scrape_error_drop_interval":"0s","scrape_timeout":"10s","scrape_timeout_offset":"500ms","warmup_delay":"0s"}}` | SQL Exporter configuration, can be a dictionary, or a template yaml string. |
| config.global.max_connections | int | `3` | Number of open connections. |
| config.global.max_idle_connections | int | `3` | Number of idle connections. |
| config.global.min_interval | string | `"0s"` | Minimum interval between collector runs. |
| config.global.scrape_error_drop_interval | string | `"0s"` | Interval between dropping scrape_errors_total metric: by default the metric is persistent. |
| config.global.scrape_timeout | string | `"10s"` | Scrape timeout |
| config.global.scrape_timeout_offset | string | `"500ms"` | Scrape timeout offset. Must be strictly positive. |
| config.global.warmup_delay | string | `"0s"` | Delay between collector scrapes during the startup cache warmup. Disabled by default. |
| target | object | `nil` | Check documentation. Mutually exclusive with `jobs`  |
| jobs   | list | `nil` | Check documentation. Mutually exclusive with `target` |
| collector_files | list | `[]` | Check documentation |

To generate the config as a part of a helm release, please set the `.Values.createConfig` to true, and define a config under the `.Values.config` property.

To configure `target`, `jobs`, `collector_files` please refer to the [documentation](https://github.com/burningalchemist/sql_exporter/blob/master/documentation/sql_exporter.yml) in the source repository. These values are not set by default.

It's also possible to define collectors (i.e. metrics and queries) in separate files, and specify the filenames in the `collector_files` list. For that we can use `CollectorFiles` field (check `values.yaml` for the available example).

## Dev Notes

After changing default `Values`, please execute `make gen_docs` to update the `README.md` file. Readme file is generated by the `helm-docs` tool, so make sure not to edit it manually.
