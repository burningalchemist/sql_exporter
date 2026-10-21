# Per-query observability metrics

When `global.enable_query_metrics` is set to true, every scrape emits two additional gauges per query in the
configuration:

- `query_duration_seconds{collector="<collector_name>", query="<query_name>"}` - wall-clock time the query took during
  the most recent scrape, including row scanning. Emitted even when the query errors, so spikes preceding a failure
  remain visible.
- `query_rows_returned{collector="<collector_name>", query="<query_name>"}` — number of rows the database returned
  during the most recent scrape. Errored or skipped rows are not counted.

Both metrics inherit the same constant labels as `up` / `scrape_duration_seconds` (notably
`target` in multi-target / jobs mode), and the `collector` label keeps query names from different collectors distinct.
The feature is off by default to keep the existing metric surface unchanged.
