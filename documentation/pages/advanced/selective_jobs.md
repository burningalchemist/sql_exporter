# Selective scraping

It is possible to pass an optional list of job names to filter out metrics as query parameters. This might be useful
for scraping targets with different intervals or any other advanced use cases, when calling all jobs at once is
undesired.

The `jobs[]` query parameter may be used multiple times. The resulted HTTP request would look like this:

```http
GET /metrics?jobs[]=db_targets1&jobs[]=db_targets2
```

In Prometheus configuration we can use this syntax under the [scrape
config](https://prometheus.io/docs/prometheus/latest/configuration/configuration/#%3Cscrape_config%3E):

```yaml
params:
  jobs[]:
    - db_targets1
    - db_targets2
```
