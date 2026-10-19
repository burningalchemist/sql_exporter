# Overview

SQL Exporter is a configuration-driven exporter that exposes metrics gathered from databases, for use by the Prometheus
monitoring system.

Per the Prometheus philosophy, scrapes are **synchronous** (i.e. metrics are collected on every `/metrics` poll).
However, to keep load at reasonable levels, minimum collection intervals may optionally be set per collector, producing
cached metrics when queried more frequently than the configured interval.

## Why it exists

Originally, SQL Exporter started off as an exporter for `Microsoft SQL Server`, for which no reliable exporters
existed.

But what is the point of a configuration-driven SQL exporter, if you're going to use it along with 2 more
exporters with wholly different world views and configurations, because you also have `MySQL` and `PostgreSQL` instances to
monitor?..

## Alternatives

Some alternative database agnostic exporters are available. They both do the collection at fixed intervals, independent
of Prometheus scrapes. This is partly a philosophical issue, but practical issues are not all that difficult to
imagine:

- jitter;
- duplicate data points;
- collected but not scraped data points.

The control they provide over which labels get applied is limited, and the base label set spammy. And finally,
configurations are not easily reused without copy-pasting and editing across jobs and instances.

## Credits

This is a fork of the original solution created by [@free](https://github.com/free/sql_exporter) that was abandoned in
2019. The development since has been taken over by [@burningalchemist](https://github.com/burningalchemist), who
maintains and supports it in their free time.

## 
