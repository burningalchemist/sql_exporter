# Supported databases

Out of the box, `sql_exporter` provides support for the following databases and compatible interfaces:

- MySQL
- PostgreSQL
- Microsoft SQL Server
- Clickhouse
- Oracle Database
- Snowflake
- Vertica

In fact, almost any database for which there is a Go driver that works via `database/sql` interface, may be monitored
after rebuilding the binary with the respective driver included.

You can find detailed instructions on how to add a custom database driver in the [Custom
drivers](../advanced/custom_drivers.md) section.
