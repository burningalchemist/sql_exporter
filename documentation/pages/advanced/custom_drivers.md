# Custom drivers

Given that `xo/dburl` module is used to parse connection strings, any database that is supported by that library can be
used with `sql_exporter`. The list of supported databases can be found in the [xo/dburl
documentation](https://github.com/xo/dburl#database-schemes-aliases-and-drivers).

If the database exists in the list but does not work, please file an issue.

In any of these cases we need to update `drivers.go` and perform blank imports of the required drivers. For example,
adding a native sqlite driver would require adding the following line:

```go
// drivers.go
import (
	_ "github.com/ClickHouse/clickhouse-go/v2"
	_ "github.com/go-sql-driver/mysql"
	_ "github.com/jackc/pgx/v5/stdlib"
	_ "github.com/lib/pq"
	_ "github.com/microsoft/go-mssqldb/azuread"
	_ "github.com/sijms/go-ora/v3"
	_ "github.com/snowflakedb/gosnowflake/v2"
	_ "github.com/vertica/vertica-sql-go"

    _ "modernc.org/sqlite" // <- this is the new driver added as a blank import
)
```

Once it's been added, we need to rebuild the binary to include the new driver. Given that Golang compiler is installed
and configured, this is as simple as running `make build` command.

It's also may be useful to reduce the size of the binary by only including specific set of drivers like Postgres, MySQL
and MSSQL. To avoid manual manipulation there is a helper code generator available, so we can run the following
commands:

```shell
make drivers-minimal
make build
```

The first command will regenerate `drivers.go` file with a minimal set of imported drivers using `drivers_gen.go`.

Running `make drivers-all` will regenerate driver set back to the current defaults.

Feel free to revisit and add more drivers as required. There's also the `custom` list that allows managing a separate
list of drivers for special needs.
