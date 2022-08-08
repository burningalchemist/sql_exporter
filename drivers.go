package sql_exporter

import (
	_ "github.com/ClickHouse/clickhouse-go" // register the ClickHouse driver
	_ "github.com/go-sql-driver/mysql"      // register the MySQL driver
	_ "github.com/jackc/pgx/v4/stdlib"      // register the pgx PostgreSQL driver
	_ "github.com/lib/pq"                   // register the libpq PostgreSQL driver
	_ "github.com/microsoft/go-mssqldb"     // register the MS-SQL driver
	_ "github.com/snowflakedb/gosnowflake"  // register the Snowflake driver
	_ "github.com/vertica/vertica-sql-go"   // register the Vertica driver
)
