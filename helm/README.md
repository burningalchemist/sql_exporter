# sql-exporter



![Version: 0.13.5](https://img.shields.io/badge/Version-0.13.5-informational?style=flat-square) ![Type: application](https://img.shields.io/badge/Type-application-informational?style=flat-square) ![AppVersion: 0.18.6](https://img.shields.io/badge/AppVersion-0.18.6-informational?style=flat-square) 

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
| service.labels | object | `{}` | Service labels |
| service.annotations | object | `{}` | Service annotations |
| ingress.enabled | bool | `false` |  |
| ingress.labels | object | `{}` | Ingress labels |
| ingress.annotations | object | `{}` | Ingress annotations |
| ingress.ingressClassName | string | `""` | Ingress class name |
| ingress.host | string | `""` | Ingress host |
| ingress.path | string | `"/"` | Ingress path |
| ingress.tls | object | `{"crt":"","enabled":false,"key":"","secretName":""}` | Ingress TLS, can be defined by cert secret, or by key and cert. |
| ingress.tls.secretName | string | `""` | Ingress tls secret if already exists. |
| ingress.tls.crt | string | `""` | Ingress tls.crt, required if you don't have secret name. |
| ingress.tls.key | string | `""` | Ingress tls.key, required if you don't have secret name. |
| extraContainers | object | `{}` | Arbitrary sidecar containers list |
| initContainers | object | `{}` | Arbitrary sidecar containers list for 1.29+ kubernetes |
| extraManifests | list | `[]` | Arbitrary manifests list |
| serviceAccount.create | bool | `true` | Specifies whether a Service Account should be created, creates "sql-exporter" service account if true, unless overriden. Otherwise, set to `default` if false, and custom service account name is not provided. Check all the available parameters. |
| serviceAccount.annotations | object | `{}` | Annotations to add to the Service Account |
| resources | object | `{}` | Resource limits and requests for the application controller pods |
| podLabels | object | `{}` | Pod labels |
| podAnnotations | object | `{}` | Pod annotations |
| podSecurityContext | object | `{}` | Pod security context |
| createConfig | bool | `true` | Set to true to create a config as a part of the helm chart |
| logLevel | string | `"info"` | Set log level (info if unset) |
| logFormat | string | `"logfmt"` | Set log format (logfmt if unset) |
| dynamicConfig | object | `{"enabled":false,"secretKey":"dsn","secretName":"","template":"global:\n  scrape_timeout: 10s\n  scrape_timeout_offset: 500ms\n  scrape_error_drop_interval: 0s\n  min_interval: 0s\n  max_connections: 3\n  max_idle_connections: 3\ntarget:\n  data_source_name: \"__TYPE__://__DSN__\"\n  collectors: []\n","type":"postgres","useApplicationName":false}` | Generate sql_exporter.yml from a secret-held partial DSN via initContainer |
| dynamicConfig.enabled | bool | `false` | Enable dynamic config generation from secret |
| dynamicConfig.secretName | string | `""` | Secret name that holds partial DSN (without scheme), e.g. user:pass@host:port/db |
| dynamicConfig.secretKey | string | `"dsn"` | Key in the secret that holds the partial DSN |
| dynamicConfig.type | string | `"postgres"` | Driver scheme to prepend (e.g. postgres, mysql, sqlserver) |
| dynamicConfig.useApplicationName | bool | `false` | Automatically add application_name parameter to DSN using chart fullname |
| dynamicConfig.template | string | `"global:\n  scrape_timeout: 10s\n  scrape_timeout_offset: 500ms\n  scrape_error_drop_interval: 0s\n  min_interval: 0s\n  max_connections: 3\n  max_idle_connections: 3\ntarget:\n  data_source_name: \"__TYPE__://__DSN__\"\n  collectors: []\n"` | Template used to write sql_exporter.yml; __TYPE__ and __DSN__ are replaced |
| webConfig | object | `{"basicAuth":{"bcryptCost":12,"enabled":false,"initFromSecret":{"enabled":false,"image":"httpd:alpine","imagePullPolicy":"IfNotPresent","secretKey":"password","secretName":""},"username":"prometheus","users":{}},"enabled":false,"template":"","tls":{"certFile":"tls.crt","certKey":"tls.crt","keyFile":"tls.key","keyKey":"tls.key","secretName":""}}` | Enable and configure Prometheus web config file support web-config.yml is automatically placed at /etc/sql_exporter/web-config.yml |
| webConfig.template | string | `""` | Template for web-config content (Exporter Toolkit format). Set to empty string to use default template (defined in _helpers.tpl) Default: TLS 1.3 with AES-GCM cipher suites, uses cert from webConfig.tls.secretName You can override with your own YAML string here if needed |
| webConfig.tls.secretName | string | `""` | Optional secret that holds tls.crt/tls.key. When set, it is mounted and used by web-config. |
| webConfig.tls.certKey | string | `"tls.crt"` | Key names within the secret for certificate and key |
| webConfig.tls.certFile | string | `"tls.crt"` | Filenames to project into the container; defaults match certKey/keyKey |
| webConfig.basicAuth.enabled | bool | `false` | Enable basic auth in web-config; passwords must be bcrypt hashes |
| webConfig.basicAuth.username | string | `"prometheus"` | Username to protect /metrics |
| webConfig.basicAuth.bcryptCost | int | `12` | Bcrypt cost used when hashing via initFromSecret |
| webConfig.basicAuth.users | object | `{}` | Map of username: bcryptHash (when not using initFromSecret) |
| webConfig.basicAuth.initFromSecret.enabled | bool | `false` | Use an initContainer to read plaintext from a secret and bcrypt it into web-config |
| webConfig.basicAuth.initFromSecret.secretName | string | `""` | Secret name containing plaintext password |
| webConfig.basicAuth.initFromSecret.secretKey | string | `"password"` | Key in the secret that contains plaintext password |
| webConfig.basicAuth.initFromSecret.image | string | `"httpd:alpine"` | Image used for bcrypt hashing (httpd:alpine has htpasswd at /usr/local/apache2/bin/htpasswd) |
| reloadEnabled | bool | `false` | Enable reload collector data handler (endpoint /reload) |


### Prometheus ServiceMonitor

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| serviceMonitor.enabled | bool | `true` | Enable ServiceMonitor |
| serviceMonitor.interval | string | `"15s"` | ServiceMonitor interval |
| serviceMonitor.path | string | `"/metrics"` | ServiceMonitor path |
| serviceMonitor.metricRelabelings | object | `{}` | ServiceMonitor metric relabelings |
| serviceMonitor.relabelings | object | `{}` | ServiceMonitor relabelings |
| serviceMonitor.namespace | string | `nil` | ServiceMonitor namespace override (default is .Release.Namespace) |
| serviceMonitor.selector | object | `{}` | Additional labels for ServiceMonitor (for Prometheus serviceMonitorSelector matching) Example: selector: { monitored: dox-prometheus } |
| serviceMonitor.scrapeTimeout | string | `nil` | ServiceMonitor scrape timeout |

### Configuration

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| config | object | `{"global":{"max_connections":3,"max_idle_connections":3,"min_interval":"0s","scrape_error_drop_interval":"0s","scrape_timeout":"10s","scrape_timeout_offset":"500ms"}}` | SQL Exporter configuration, can be a dictionary, or a template yaml string. |
| config.global.scrape_timeout | string | `"10s"` | Scrape timeout |
| config.global.scrape_timeout_offset | string | `"500ms"` | Scrape timeout offset. Must be strictly positive. |
| config.global.scrape_error_drop_interval | string | `"0s"` | Interval between dropping scrape_errors_total metric: by default the metric is persistent. |
| config.global.min_interval | string | `"0s"` | Minimum interval between collector runs. |
| config.global.max_connections | int | `3` | Number of open connections. |
| config.global.max_idle_connections | int | `3` | Number of idle connections. |
| target | object | `nil` | Check documentation. Mutually exclusive with `jobs`  |
| jobs   | list | `nil` | Check documentation. Mutually exclusive with `target` |
| collector_files | list | `[]` | Check documentation |

To generate the config as a part of a helm release, please set the `.Values.createConfig` to true, and define a config under the `.Values.config` property.

To configure `target`, `jobs`, `collector_files` please refer to the [documentation](https://github.com/burningalchemist/sql_exporter/blob/master/documentation/sql_exporter.yml) in the source repository. These values are not set by default.

It's also possible to define collectors (i.e. metrics and queries) in separate files, and specify the filenames in the `collector_files` list. For that we can use `CollectorFiles` field (check `values.yaml` for the available example).

### Using application_name with dynamicConfig

When using `dynamicConfig` to generate DSN from secrets, you can automatically add the `application_name` parameter to identify connections in PostgreSQL. This is useful for DBAs to track which SQL statements are issued by sql_exporter.

```yaml
dynamicConfig:
  enabled: true
  secretName: "my-postgres-dsn"
  secretKey: "dsn"
  type: "postgres"
  useApplicationName: true  # Automatically adds application_name to DSN
```

When `useApplicationName: true`, the chart will automatically append `application_name=<release-name>-sql-exporter` to your DSN. The application name will be visible in PostgreSQL's `pg_stat_activity` view and logs, making it easy to identify connections from this exporter instance.

Example: If your release name is `myapp`, DBAs will see connections with `application_name: myapp-sql-exporter` in `pg_stat_activity`.

## Dev Notes

After changing default `Values`, please execute `make gen_docs` to update the `README.md` file. Readme file is generated by the `helm-docs` tool, so make sure not to edit it manually.
