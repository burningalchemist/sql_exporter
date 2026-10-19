# Features

- **Database-agnostic with broad driver support**. The exporter supports a wide range of databases including
  PostgreSQL, MySQL, Microsoft SQL Server, and more.

- **Scrape-synchronous collection with optional caching**.
  Metrics are collected on each Prometheus `/metrics` poll, which avoids the jitter, duplicate points and unscraped data
  of fixed-interval exporters. Minimal interval can be set per collector to serve cached results when scrapes are more
  frequent.

- **Staggered cache warmup**. To prevent thundering herd effect on the database, the cache warmup is staggered on
  startup if configured.

- **Fully configuration-driven metrics**.
  Metrics and the SQL queries behind them are defined in YAML. Queries are grouped into reusable collectors.

- **Flexible metric mapping**.
  Each metric supports a flexible mapping from query results to Prometheus metrics to enrich the data with labels and
  values. The exporter can be configured to skip empty rows or rows with `NULL` label values.

- **Multi-target (jobs) mode**. Single exporter instance can monitor many databases through a jobs list. Each job has
  named targets, shared labels and its own collector set.

- **Selective job scraping**. The `jobs[]` query parameter lets Prometheus scrape particular jobs. This allows for more
  flexible scraping strategies.

- **Secret manager integration** Connection string can reference a secret instead of holding static credentials for the
  following secret managers:

  - Kubernetes Secrets
  - AWS Secrets Manager
  - Google Cloud Secrets Manager
  - Hashicorp Vault

- **Secured HTTP/S endpoint**.
  Use TLS certificates and basic authentication to secure the endpoint.

- **Request rate limiting**.
  Limit the number of requests per second to avoid overload and protect the exporter from DoS attacks.

- **Live reload of configuration without restart**. Reload the configuration file on SIGHUP signal or via HTTP endpoint
  without restarting the exporter.

- **Automatic metrics**. Extra metrics for configured targets are collected automatically, including scrape duration,
  scrape errors, etc.

- **Builtin validation and config check**.
