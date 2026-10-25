# General

When working with `sql_exporter` there is a common workflow that involves creating a configuration file and collector
files. The configuration file is typically named `sql_exporter.yml` and contains the following sections:

- defined targets (i.e. database connection strings);
- collectors (may be defined inline, in separate files, or both);
- metrics and queries.

The exporter reads this configuration file at startup and uses it to connect to the database and collect metrics.

## Collectors

Collector files as separate files are provided via `collector_files` configuration option. This allows users to
organize their collectors in a more modular way, making it easier to manage and maintain them. The `collector_files`
option can accept a list of filenames or glob patterns, allowing users to load multiple collector files at once.

The exporter will look for collector files with the `.yml` extension and load them automatically. Users might provide
static filenames or glob patterns to load multiple collector files at once. For example, `collectors/*.yml` will load
all `.yml` files in the `collectors` directory.

A rule of thumb when using multiple collector files is to organize them within folders, so globbing becomes a helper
and not a burden. For example, you can have a `collectors` folder with subfolders for different types of collectors,
and then use glob patterns to load them all at once.

Reference link: [Collectors](../config.html#collectors)

## Targets

### Single target mode

The single target mode is the simplest way to run the exporter. In this mode, the exporter runs in single target mode
(`target`), which means it connects to a single database instance and collects metrics from it. It's a straightforward
way to evaluate the exporter and its capabilities, and whether it suits your needs.

This mode makes sense if sql_exporter is deployed as a sidecar to a single database instance, or if you have a single
database instance that you want to monitor.

Reference link: [Single target mode](../config.html#target-single-target-mode)

### Multiple target mode

It's also possible to run a single exporter instance against multiple database connections. To do so we use `jobs`
(multiple target mode) list instead of the `target` (single target mode) section. Multiple jobs may be organized to
have different parameters, such as:

- defined targets;
- assigned collectors;
- attached labels.

Eventually, `sql_exporter` will run multiple jobs in parallel, each with its own configuration and connection settings
and compiling metrics from different database instances into a single metrics export to be scraped by Prometheus.

Reference link: [Multiple target mode](../config.html#jobs-multiple-target-mode)

## Metrics and Queries

The metrics and queries section of the configuration file defines the metrics that will be collected from the database
and how they will be exposed to Prometheus.

Each metric consists of a name, a query, respective mapping of the query result to the metric, and optional labels. The
query is a SQL statement that retrieves the desired result set from the database. The mapping defines how the columns
of the result set are mapped to the metric's labels and values. Optional labels can be added to provide additional
context to the metric.

> [!IMPORTANT]
> Since mapping of the query result to the metric is done via column names, it is important to ensure that the query
> returns the expected columns with the **correct names**.
>
> Otherwise, the metric may not be collected correctly or may be dropped entirely.

Metrics can be defined to return a single or multiple values from a query. For multiple values (`values` field) it is
also required to provide `value_label` parameter in the mapping section, which specifies the arbitrary label name that
will be used to differentiate between the different values returned by the query. The value for this label will be
inferred from the column name of the query result. Check [Complete example](../config.html#complete-example) for
clarification.

To populate labels for metric identification, the query needs to return additional columns that will be mapped to
labels in the metric via `key_labels` mapping.

Reference link: [Metric](../config.html#metric)
