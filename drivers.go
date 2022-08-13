package sql_exporter

import (
	_ "github.com/go-sql-driver/mysql"  // register the MySQL driver
	_ "github.com/lib/pq"               // register the libpq PostgreSQL driver
	_ "github.com/microsoft/go-mssqldb" // register the MS-SQL driver
)
