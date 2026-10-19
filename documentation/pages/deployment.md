# Deployment

SQL Exporter is deployed alongside the DB server it collects metrics from. If both the exporter and the DB server are
on the same host, they will share the same failure domain: they will usually be either both up and running or both
down.

When the database is unreachable, `/metrics` responds with HTTP code `500 Internal Server Error`, causing Prometheus to
record `up=0` for that scrape. Only metrics defined by collectors are exported on the `/metrics` endpoint.

SQL Exporter own process metrics are exported at `/sql_exporter_metrics`.

