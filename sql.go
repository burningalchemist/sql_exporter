package sql_exporter

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	_ "github.com/ClickHouse/clickhouse-go" // register the ClickHouse driver
	_ "github.com/denisenkom/go-mssqldb"    // register the MS-SQL driver
	_ "github.com/go-sql-driver/mysql"      // register the MySQL driver
	_ "github.com/jackc/pgx/v4/stdlib"      // register the pgx PostgreSQL driver
	_ "github.com/lib/pq"                   // register the libpq PostgreSQL driver
	_ "github.com/snowflakedb/gosnowflake"  // register the Snowflake driver
	_ "github.com/vertica/vertica-sql-go"   // register the Vertica driver

	"k8s.io/klog/v2"
)

// OpenConnection extracts the driver name from the DSN (expected as the URI scheme), adjusts it where necessary (e.g.
// some driver supported DSN formats don't include a scheme), opens a DB handle ensuring early termination if the
// context is closed (this is actually prevented by `database/sql` implementation), sets connection limits and returns
// the handle.
//
// Below is the list of supported databases (with built in drivers) and their DSN formats. Unfortunately there is no
// dynamic way of loading a third party driver library (as e.g. with Java classpaths), so any driver additions require
// a binary rebuild.
//
// MySQL
//
// Using the https://github.com/go-sql-driver/mysql driver, DSN format (passed to the driver stripped of the `mysql://`
// prefix):
//   mysql://username:password@protocol(host:port)/dbname?param=value
//
// PostgreSQL (libpq driver)
//
// Using the https://godoc.org/github.com/lib/pq driver, DSN format (passed through to the driver unchanged):
//   postgres://username:password@host:port/dbname?param=value
//
// PostgreSQL (pgx driver)
//
// Using the https://godoc.org/github.com/lib/pq driver, DSN format (passed through to the driver unchanged):
//   pgx://username:password@host:port/dbname?param=value
//
///
// MS SQL Server
//
// Using the https://github.com/denisenkom/go-mssqldb driver, DSN format (passed through to the driver unchanged):
//   sqlserver://username:password@host:port/instance?param=value
//
// Clickhouse
//
// Using the https://github.com/kshvakov/clickhouse driver, DSN format (passed to the driver with the`clickhouse://`
// prefix replaced with `tcp://`):
//   clickhouse://host:port?username=username&password=password&database=dbname&param=value
//
// Snowflake
//
// Using the https://godoc.org/github.com/snowflakedb/gosnowflake driver, DSN format (passed to the driver stripped
// of the `snowflake://`` prefix):
//   snowflake://username:password@account/dbname?role=rolename&warehouse=warehousename&param=value
//
// Vertica
//
// Using the https://github.com/vertica/vertica-sql-go driver, DSN format (passed through to the driver unchanged):
//   vertica://user:password@host:port/dbname?param=value
//
func OpenConnection(ctx context.Context, logContext, dsn string, maxConns, maxIdleConns int, maxConnLifetime time.Duration) (*sql.DB, error) {
	// Extract driver name from DSN.
	idx := strings.Index(dsn, "://")
	if idx == -1 {
		return nil, fmt.Errorf("missing driver in data source name. Expected format `<driver>://<dsn>`")
	}
	driver := dsn[:idx]

	// Adjust DSN, where necessary.
	switch driver {
	case "mysql":
		dsn = strings.TrimPrefix(dsn, "mysql://")
	case "clickhouse":
		dsn = "tcp://" + strings.TrimPrefix(dsn, "clickhouse://")
	case "snowflake":
		dsn = strings.TrimPrefix(dsn, "snowflake://")
	case "pgx":
		dsn = "postgres://" + strings.TrimPrefix(dsn, "pgx://")
	}

	// Open the DB handle in a separate goroutine so we can terminate early if the context closes.
	var (
		conn *sql.DB
		err  error
		ch   = make(chan error)
	)
	go func() {
		conn, err = sql.Open(driver, dsn)
		close(ch)
	}()
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-ch:
		if err != nil {
			return nil, err
		}
	}

	conn.SetMaxIdleConns(maxIdleConns)
	conn.SetMaxOpenConns(maxConns)
	conn.SetConnMaxLifetime(maxConnLifetime)

	if klog.V(1).Enabled() {
		if len(logContext) > 0 {
			logContext = fmt.Sprintf("[%s] ", logContext)
		}
		klog.Infof("%sDatabase handle successfully opened with '%s' driver", logContext, driver)
	}
	return conn, nil
}

// PingDB is a wrapper around sql.DB.PingContext() that terminates as soon as the context is closed.
//
// sql.DB does not actually pass along the context to the driver when opening a connection (which always happens if the
// database is down) and the driver uses an arbitrary timeout which may well be longer than ours. So we run the ping
// call in a goroutine and terminate immediately if the context is closed.
func PingDB(ctx context.Context, conn *sql.DB) error {
	ch := make(chan error, 1)

	go func() {
		ch <- conn.PingContext(ctx)
		close(ch)
	}()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-ch:
		return err
	}
}
