# Connection Strings

> [!IMPORTANT]
> If your connection string contains special characters in any part of your connection string (including passwords),
> you might need to apply [URL encoding](https://en.wikipedia.org/wiki/URL_encoding#Reserved_characters)
> (percent-encoding) to them.
>
> For example, `p@$$w0rd#abc` then becomes `p%40%24%24w0rd%23abc`.

To keep things simple and yet allow fully configurable database connections, SQL Exporter uses database URLs
(URL-format connection strings) (like `sqlserver://user:pass@dbserver1.example.com:1433`) to refer to database
instances.

This exporter relies on `xo/dburl` [package](https://github.com/xo/dburl#example-urls) for parsing URL-format connection strings. It allows to have a unified way
to specify connection strings across different database drivers without having to worry about the specific connection
string format for each database. This makes adding support for new databases easy, and can be handled by the user
accordingly.

However, this can potentially affect your connection to certain databases like MySQL, which comes with their own
format, so you might want to adjust your connection string accordingly.

```text
mysql://user:pass@localhost/dbname - for TCP connection
mysql:/var/run/mysqld/mysqld.sock - for Unix socket connection
```
